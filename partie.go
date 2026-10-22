package main

import (
	"sync"
	"time"

	"aeropostale/jeu"
)

// Le moteur joue la partie dans sa propre goroutine, en tenant le verrou de
// la Partie ; il le relâche pendant les pauses d'animation et quand il attend
// une décision du joueur humain. L'interface (Update/Draw) prend ce verrou
// pour lire l'état et poster les réponses.

type TypeReq int

const (
	ReqAucune   TypeReq = iota // l'ordinateur joue
	ReqEquiper                 // acheter, construire, ou passer
	ReqDevis                   // construire la piste au prix annoncé ?
	ReqAvion                   // choisir l'avion à déplacer
	ReqDecoller                // jet d'autonomie lancé : décoller ou rester
	ReqVoler                   // en vol : case suivante ou atterrissage
	ReqCrash                   // jet de panne 3 : case du crash
	ReqEscale                  // posé sur sa piste : entreposer, reprendre
)

type Requete struct {
	Type   TypeReq
	Avion  *jeu.Avion
	Case   jeu.Case // devis : case de la piste
	Prix   int
	Choix  []jeu.Case
	Avions []*jeu.Avion
}

// commandes de l'interface
type (
	cmdAcheter    struct{ t jeu.TypeAvion }
	cmdPiste      struct{ k jeu.Case }
	cmdOui        struct{}
	cmdNon        struct{}
	cmdAvion      struct{ a *jeu.Avion }
	cmdVol        struct{ dest jeu.Case }
	cmdAtterrir   struct{}
	cmdCase       struct{ k jeu.Case }
	cmdEntreposer struct{}
	cmdReprendre  struct{}
	cmdFin        struct{}
)

type Partie struct {
	mu      sync.Mutex
	E       *jeu.Etat
	Humain  int
	Req     Requete
	rep     chan any
	Erreur  string
	Rapide  bool
	Termine bool
	auto    int     // tours joués par l'ordinateur pour le joueur humain (captures)
	arret   TypeReq // captures : l'ordinateur joue pour l'humain jusqu'à cette demande
	muet    bool    // pas de pause d'animation (tests)
}

func nouvellePartie(couleur, nbAdv int, graine uint64, toursAuto int) *Partie {
	p := &Partie{Humain: 0, rep: make(chan any, 1), auto: toursAuto}
	noms := []string{"Vous"}
	couleurs := []int{couleur}
	ctrls := []jeu.Controleur{&Humain{p: p, ia: jeu.NouvelleIA(0.5)}}
	pilotes := []string{"Mermoz", "Guillaumet", "Saint-Exupéry", "Daurat"}
	for c := range 4 {
		if c != couleur && len(noms) <= nbAdv {
			noms = append(noms, pilotes[len(noms)-1])
			couleurs = append(couleurs, c)
			ctrls = append(ctrls, jeu.NouvelleIA(0.25+0.25*float64(len(noms)-1)))
		}
	}
	p.E = jeu.Nouveau(noms, couleurs, ctrls, graine)
	p.E.Humain = p.Humain
	p.E.Pause = p.pause
	return p
}

// lancer : la partie se joue en arrière-plan.
func (p *Partie) lancer() {
	go func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		for !p.E.Fini {
			p.E.JouerTour()
		}
		p.Termine = true
	}()
}

// jouerAuto : premiers tours joués d'un coup (options de capture).
func (p *Partie) jouerAuto() {
	for p.E.Tour < p.auto && !p.E.Fini {
		p.E.JouerTour()
	}
}

func (p *Partie) pause(pas bool) {
	if p.E.Tour <= p.auto || p.muet || p.arret != ReqAucune {
		return
	}
	d := 650 * time.Millisecond
	switch {
	case pas && p.E.Courant == p.Humain:
		d = 110 * time.Millisecond
	case pas:
		d = 190 * time.Millisecond
	case p.E.Courant == p.Humain:
		d = 350 * time.Millisecond
	}
	if p.Rapide {
		d /= 5
	}
	p.mu.Unlock()
	time.Sleep(d)
	p.mu.Lock()
}

// attendre : publie une demande et attend la réponse de l'interface.
func (p *Partie) attendre(r Requete) any {
	p.Req = r
	p.mu.Unlock()
	v := <-p.rep
	p.mu.Lock()
	p.Req = Requete{}
	return v
}

// repondre : appelé par l'interface, verrou tenu.
func (p *Partie) repondre(v any) {
	if p.Req.Type == ReqAucune {
		return
	}
	p.Req = Requete{}
	p.Erreur = ""
	select {
	case p.rep <- v:
	default:
	}
}

// Humain : contrôleur du joueur humain ; délègue à l'IA pendant les tours automatiques.
type Humain struct {
	p      *Partie
	ia     *jeu.IA
	file   []jeu.Case // trajet choisi d'un clic, parcouru case par case
	poser  bool       // se poser au bout du trajet
	sursis bool       // déjà prévenu que l'autonomie est épuisée
}

func (h *Humain) auto(e *jeu.Etat) bool { return e.Tour <= h.p.auto }

// passe : captures, l'ordinateur joue pour l'humain tant que la demande
// attendue (option -arret) n'est pas arrivée.
func (h *Humain) passe(t TypeReq) bool {
	if h.p.arret == ReqAucune {
		return false
	}
	if h.p.arret == t {
		h.p.arret = ReqAucune
		return false
	}
	return true
}

func (h *Humain) erreur(err error) {
	if err != nil {
		h.p.Erreur = err.Error()
	} else {
		h.p.Erreur = ""
	}
}

func (h *Humain) Equiper(e *jeu.Etat, j int) {
	if h.auto(e) || h.passe(ReqEquiper) {
		h.ia.Equiper(e, j)
		return
	}
	for {
		switch c := h.p.attendre(Requete{Type: ReqEquiper}).(type) {
		case cmdAcheter:
			_, err := e.Acheter(j, c.t)
			h.erreur(err)
		case cmdPiste:
			prix, err := e.DevisPiste(j, c.k)
			if err != nil {
				h.erreur(err)
				continue
			}
			if _, ok := h.p.attendre(Requete{Type: ReqDevis, Case: c.k, Prix: prix}).(cmdOui); ok {
				h.erreur(e.Construire(j, c.k, prix))
			} else {
				e.Log("%s renonce à construire.", e.Nom(j))
			}
		case cmdFin:
			return
		}
	}
}

func (h *Humain) ChoisirAvion(e *jeu.Etat, j int, restants []*jeu.Avion) *jeu.Avion {
	if h.auto(e) || h.passe(ReqAvion) {
		return h.ia.ChoisirAvion(e, j, restants)
	}
	for {
		switch c := h.p.attendre(Requete{Type: ReqAvion, Avions: restants}).(type) {
		case cmdAvion:
			return c.a
		case cmdFin:
			return nil
		}
	}
}

// trajet : chemin le moins cher vers dest, budget en demi-points.
func (h *Humain) trajet(e *jeu.Etat, a *jeu.Avion, dest jeu.Case, budget int) bool {
	cout, prec := e.Portee(a.Case, budget)
	if !dest.Valide() || dest == a.Case || cout[dest.R][dest.C] < 0 {
		return false
	}
	h.file = jeu.Chemin(prec, a.Case, dest)
	h.poser = e.PeutAtterrir(a, dest)
	return true
}

func (h *Humain) Decoller(e *jeu.Etat, a *jeu.Avion) bool {
	h.file, h.poser, h.sursis = nil, false, false
	if h.auto(e) || h.passe(ReqDecoller) {
		return h.ia.Decoller(e, a)
	}
	for {
		switch c := h.p.attendre(Requete{Type: ReqDecoller, Avion: a}).(type) {
		case cmdVol:
			if h.trajet(e, a, c.dest, a.Carburant-2) {
				return true
			}
			h.p.Erreur = "Destination hors de portée."
		case cmdNon:
			return false
		}
	}
}

func (h *Humain) Voler(e *jeu.Etat, a *jeu.Avion) jeu.Ordre {
	if h.auto(e) || (h.p.arret != ReqAucune && h.p.arret != ReqVoler) {
		return h.ia.Voler(e, a)
	}
	if h.passe(ReqVoler) {
		return h.ia.Voler(e, a)
	}
	// l'autonomie vient de s'épuiser en vol : on s'arrête pour prévenir
	if a.Conso >= a.Jet && !h.sursis && !e.PeutSePoser(a) {
		h.sursis = true
		h.file, h.poser = nil, false
	}
	if len(h.file) > 0 {
		k := h.file[0]
		h.file = h.file[1:]
		return jeu.Ordre{Vers: k}
	}
	if h.poser && e.PeutSePoser(a) {
		h.poser = false
		return jeu.Ordre{Atterrir: true}
	}
	for {
		switch c := h.p.attendre(Requete{Type: ReqVoler, Avion: a}).(type) {
		case cmdVol:
			if c.dest == a.Case && e.PeutSePoser(a) {
				return jeu.Ordre{Atterrir: true}
			}
			if h.trajet(e, a, c.dest, a.Carburant) {
				k := h.file[0]
				h.file = h.file[1:]
				return jeu.Ordre{Vers: k}
			}
			h.p.Erreur = "Destination hors de portée."
		case cmdAtterrir:
			return jeu.Ordre{Atterrir: true}
		}
	}
}

func (h *Humain) CaseDeCrash(e *jeu.Etat, a *jeu.Avion, choix []jeu.Case) jeu.Case {
	if h.auto(e) || h.passe(ReqCrash) {
		return h.ia.CaseDeCrash(e, a, choix)
	}
	for {
		if c, ok := h.p.attendre(Requete{Type: ReqCrash, Avion: a, Choix: choix}).(cmdCase); ok {
			return c.k
		}
	}
}

func (h *Humain) Escale(e *jeu.Etat, a *jeu.Avion) {
	if h.auto(e) || h.passe(ReqEscale) {
		h.ia.Escale(e, a)
		return
	}
	p := e.PisteDe(a.Joueur, a.Case)
	for len(a.Sacs) > 0 || len(p.Stock) > 0 {
		switch h.p.attendre(Requete{Type: ReqEscale, Avion: a}).(type) {
		case cmdEntreposer:
			h.erreur(e.Entreposer(a))
		case cmdReprendre:
			h.erreur(e.Reprendre(a))
		default:
			return
		}
	}
}
