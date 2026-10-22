// Aéropostale — d'après le jeu de François Nédélec paru en encart dans
// Jeux & Stratégie (1983, dessin Claude Lacroix). Un joueur humain contre une
// à trois compagnies aériennes menées par l'ordinateur.
package main

import (
	"flag"
	"log"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"aeropostale/jeu"
)

const (
	LargeurEcran = 1600
	HauteurEcran = 1000
	PanneauX     = 1212.0
	LargPanneau  = LargeurEcran - PanneauX - 12
)

type Ecran int

const (
	EcranAccueil Ecran = iota
	EcranJeu
)

type Game struct {
	ecran   Ecran
	couleur int
	nbAdv   int
	p       *Partie

	// saisie en cours
	modePiste bool     // le prochain clic choisit la case d'une piste
	selAvion  int      // avion dont on montre la feuille de route (-1 : celui du moment)
	survol    jeu.Case // case sous la souris

	regles       bool
	pageRegle    int
	accueilRegle bool

	boutons []*Bouton
	mx, my  int
	images  int
	t0      time.Time

	// animation et affichage
	glisse   map[int]*glissement
	caps     map[int]float64
	dernier  *jeu.Evenement
	depuis   time.Time
	zone     zoneVol
	yBoutons float64
	yFeuille float64
}

func nouveauJeu() *Game {
	g := &Game{couleur: 0, nbAdv: 2, t0: time.Now()}
	g.reinitSaisie()
	return g
}

func (g *Game) reinitSaisie() {
	g.modePiste = false
	g.selAvion = -1
}

func (g *Game) Layout(int, int) (int, int) { return LargeurEcran, HauteurEcran }

func (g *Game) commencer(graine uint64, toursAuto int) {
	if graine == 0 {
		graine = rand.Uint64()
	}
	g.p = nouvellePartie(g.couleur, g.nbAdv, graine, toursAuto)
	g.p.arret = arretCapture
	g.p.jouerAuto()
	g.p.lancer()
	g.ecran = EcranJeu
	g.glisse = map[int]*glissement{}
	g.caps = map[int]float64{}
	g.reinitSaisie()
}

var dernierReq TypeReq = -1

func (g *Game) Update() error {
	g.mx, g.my = ebiten.CursorPosition()
	clic := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if g.p != nil {
		g.p.mu.Lock()
		defer g.p.mu.Unlock()
		if g.p.Req.Type != dernierReq {
			dernierReq = g.p.Req.Type
			g.modePiste = *optPiste && dernierReq == ReqEquiper
		}
	}
	g.construireBoutons()
	if clic {
		for _, b := range g.boutons {
			if b.Actif && b.Contient(g.mx, g.my) {
				b.Action()
				return nil
			}
		}
	}
	if g.ecran == EcranAccueil {
		if g.accueilRegle {
			g.touchesRegle()
		}
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.regles = !g.regles
	}
	if g.regles {
		g.touchesRegle()
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.p.Rapide = !g.p.Rapide
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.reinitSaisie()
	}
	g.survol = caseSous(float64(g.mx), float64(g.my))
	g.survolCapture()
	g.clicCapture()
	if clic && g.survol.Valide() {
		g.clicPlateau(g.survol)
	}
	return nil
}

// clicPlateau : le clic sur une case selon la saisie attendue.
func (g *Game) clicPlateau(k jeu.Case) {
	p := g.p
	e := p.E
	r := p.Req
	switch r.Type {
	case ReqEquiper:
		if g.modePiste {
			if err := e.PeutConstruire(p.Humain, k); err != nil {
				p.Erreur = err.Error()
				return
			}
			g.modePiste = false
			p.repondre(cmdPiste{k})
			return
		}
	case ReqAvion:
		for _, a := range r.Avions {
			if a.Case == k {
				p.repondre(cmdAvion{a})
				g.selAvion = -1
				return
			}
		}
	case ReqDecoller, ReqVoler:
		a := r.Avion
		if k == a.Case {
			if r.Type == ReqVoler && e.PeutSePoser(a) {
				p.repondre(cmdAtterrir{})
			}
			return
		}
		z := g.zoneDe(e, r)
		if z.cout[k.R][k.C] > 0 {
			p.repondre(cmdVol{k})
		} else {
			p.Erreur = "Cette case est hors de portée : pas assez de carburant."
		}
		return
	case ReqCrash:
		for _, c := range r.Choix {
			if c == k {
				p.repondre(cmdCase{k})
				return
			}
		}
		p.Erreur = "Choisissez l'une des six cases voisines, cerclées de rouge."
		return
	}
	// sinon : on montre la feuille de route de l'avion cliqué
	for _, a := range e.Avions {
		if a.Actif() && a.Case == k {
			g.selAvion = a.ID
			return
		}
	}
}

func (g *Game) Draw(dst *ebiten.Image) {
	if g.p != nil {
		g.p.mu.Lock()
		defer g.p.mu.Unlock()
	}
	dst.Fill(coulFond)
	switch g.ecran {
	case EcranAccueil:
		g.dessinerAccueil(dst)
		if g.accueilRegle {
			g.dessinerRegles(dst)
		}
	case EcranJeu:
		g.dessinerPlateau(dst)
		g.dessinerPanneau(dst)
		if g.regles {
			g.dessinerRegles(dst)
		} else if g.p.Termine || g.p.E.Fini {
			g.dessinerFin(dst)
		}
	}
	for _, b := range g.boutons {
		dessinerBouton(dst, b, b.Actif && b.Contient(g.mx, g.my))
	}
	g.capturer(dst)
}

func (g *Game) bouton(x, y, w, h float64, t string, actif bool, f func()) *Bouton {
	b := &Bouton{X: x, Y: y, W: w, H: h, Texte: t, Actif: actif, Action: f}
	g.boutons = append(g.boutons, b)
	return b
}

func (g *Game) construireBoutons() {
	g.boutons = g.boutons[:0]
	switch g.ecran {
	case EcranAccueil:
		g.boutonsAccueil()
	case EcranJeu:
		if g.regles {
			g.boutonsRegles()
			return
		}
		if g.p.Termine || g.p.E.Fini {
			b := g.bouton(PlateauX+LargPlat/2-160, 590, 320, 54, "NOUVELLE PARTIE", true, func() {
				c, n := g.couleur, g.nbAdv
				*g = *nouveauJeu()
				g.couleur, g.nbAdv = c, n
			})
			b.Style = BoutonGros
			return
		}
		g.boutonsJeu()
	}
}

var optPleinEcran = flag.Bool("fullscreen", false, "démarre en plein écran (F11 bascule)")

func main() {
	flag.Parse()
	chargerRessources()
	g := nouveauJeu()
	g.preparer()
	ebiten.SetWindowSize(LargeurEcran, HauteurEcran)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Aéropostale")
	ebiten.SetFullscreen(*optPleinEcran)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
