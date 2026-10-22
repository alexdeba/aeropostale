package jeu

import (
	"fmt"
	"math/rand/v2"
	"slices"
)

// Les points de déplacement se comptent par demi-points (vent arrière : ½ point par case).

type TypeAvion uint8

const (
	MoyenCourrier TypeAvion = iota
	LongCourrier
	Hydravion
	NbTypes
)

// Caracteristiques d'un type d'avion (règle : prix, plein, dés d'autonomie)
// et nombre de pions de chaque type dans l'encart.
type Caracteristiques struct {
	Nom, Court   string
	Prix         int // francs
	PrixPlein    int // francs
	Plein        int // points de déplacement
	DesAutonomie int
	Pions        int
}

var Types = [NbTypes]Caracteristiques{
	{"moyen-courrier", "M-C", 15000, 1000, 6, 1, 6},
	{"long-courrier", "L-C", 20000, 3000, 12, 2, 5},
	{"hydravion", "H", 25000, 1500, 8, 2, 4},
}

func (t TypeAvion) Car() Caracteristiques { return Types[t] }
func (t TypeAvion) String() string        { return Types[t].Nom }

const (
	CapitalDepart = 10000
	PionsPistes   = 7  // aérodromes par couleur
	SacsVictoire  = 13 // sacs à rapporter à Paris
	ToursMax      = 99 // garde-fou : au-delà, le plus de sacs l'emporte
)

// Sac de courrier : ville d'origine et tour où il a été pris.
type Sac struct {
	Ville, Tour int
}

// Ligne de la feuille de route d'un avion (tableau 1 de la règle).
type Ligne struct {
	Tour  int
	Texte string
}

type EtatAvion uint8

const (
	AuSol EtatAvion = iota
	EnVol
	Detruit
)

type Avion struct {
	ID         int
	Type       TypeAvion
	Joueur     int
	Numero     int // numéro de l'avion pour son joueur (feuille de route)
	Case       Case
	Etat       EtatAvion
	Carburant  int  // demi-points restants
	Jet        int  // jet d'autonomie en cours, en demi-points (0 : pas encore lancé)
	Conso      int  // demi-points consommés depuis le décollage
	Sursis     bool // autonomie épuisée en vol : un jet de panne par case
	Veteran    bool
	Reparation int // tour jusqu'auquel l'avion reste cloué au sol (panne 4)
	Sacs       []Sac
	PrisCeTour []int // villes où l'avion a déjà pris un sac ce tour
	Feuille    []Ligne
	Depart     Case   // case de décollage du vol en cours
	Trajet     []Case // cases traversées depuis le dernier décollage
}

func (a *Avion) Actif() bool { return a.Etat != Detruit }

func (a *Avion) Nom() string { return fmt.Sprintf("%s n° %d", a.Type.Car().Court, a.Numero) }

// Piste privée (aérodrome) d'un joueur.
type Piste struct {
	Joueur int
	Case   Case
	Stock  []Sac // courrier entreposé
}

type Joueur struct {
	Nom       string
	Couleur   int // 0 rouge, 1 jaune, 2 vert, 3 bleu (couleurs des pions)
	Capital   int
	Rapportes int // sacs rapportés à Paris
	Gains     int // francs gagnés par le courrier
	Veterans  int // pilotes vétérans qui attendent un avion à Paris
	Pilotes   int // pilotes morts
	Bloque    bool
	Numeros   int // avions achetés (numérotation)
}

// Remise d'un sac à Paris pendant le tour, pour attribuer la prime de la
// première liaison à la fin du tour.
type Remise struct {
	Joueur, Ville, TourPris, Ordre int
}

type Phase uint8

const (
	PhaseMeteo Phase = iota
	PhaseEquipement
	PhaseDeplacement
	PhaseRavitaillement
	PhaseFin
)

var nomsPhase = [...]string{"météo", "équipement", "déplacement", "ravitaillement", "fin"}

func (p Phase) String() string { return nomsPhase[p] }

// Évènement marquant (affiché par l'interface : dés, pannes, crashs).
type Evenement struct {
	Tour   int
	Joueur int
	Texte  string
	Des    []int
	Case   Case
	Grave  bool
}

type Etat struct {
	P           *Plateau
	Rng         *rand.Rand
	Joueurs     []Joueur
	Ctrl        []Controleur
	Avions      []*Avion
	Pistes      []*Piste
	Tour        int
	Vent        int
	Ordre       []int
	Courant     int
	Phase       Phase
	Actif       int    // avion en cours de déplacement (-1 sinon)
	Reliees     []bool // villes déjà reliées à Paris (prime versée)
	Remises     []Remise
	Fini        bool
	Gagnant     int
	Journal     []string
	Dernier     *Evenement
	Humain      int
	Pause       func(pas bool) // pause d'animation (l'interface relâche son verrou)
	tentee      bool           // une construction déjà tentée ce tour
	ordreRemise int
}

func Nouveau(noms []string, couleurs []int, ctrls []Controleur, graine uint64) *Etat {
	e := &Etat{
		P:       NouveauPlateau(),
		Rng:     rand.New(rand.NewPCG(graine, graine^0x9e3779b97f4a7c15)),
		Ctrl:    ctrls,
		Actif:   -1,
		Gagnant: -1,
		Pause:   func(bool) {},
	}
	e.Reliees = make([]bool, len(e.P.Villes))
	for j, n := range noms {
		e.Joueurs = append(e.Joueurs, Joueur{Nom: n, Couleur: couleurs[j], Capital: CapitalDepart})
		e.nouvelAvion(j, MoyenCourrier)
	}
	n := len(noms)
	d := e.Rng.IntN(n)
	for k := range n {
		e.Ordre = append(e.Ordre, (d+k)%n)
	}
	return e
}

func (e *Etat) Log(f string, a ...any) {
	e.Journal = append(e.Journal, fmt.Sprintf(f, a...))
}

// evenement : journal + dernier évènement marquant (affiché en grand).
func (e *Etat) evenement(j int, k Case, grave bool, des []int, f string, a ...any) {
	t := fmt.Sprintf(f, a...)
	e.Log("%s", t)
	e.Dernier = &Evenement{Tour: e.Tour, Joueur: j, Texte: t, Des: des, Case: k, Grave: grave}
}

func (e *Etat) De() int { return 1 + e.Rng.IntN(6) }

func (e *Etat) Nom(j int) string { return e.Joueurs[j].Nom }

func (e *Etat) noter(a *Avion, f string, x ...any) {
	t := fmt.Sprintf(f, x...)
	if n := len(a.Feuille); n > 0 && a.Feuille[n-1].Tour == e.Tour {
		a.Feuille[n-1].Texte += " ; " + t
		return
	}
	a.Feuille = append(a.Feuille, Ligne{e.Tour, t})
}

func (e *Etat) nouvelAvion(j int, t TypeAvion) *Avion {
	jr := &e.Joueurs[j]
	jr.Numeros++
	a := &Avion{
		ID: len(e.Avions), Type: t, Joueur: j, Numero: jr.Numeros,
		Case: e.P.CaseParis(), Carburant: 2 * t.Car().Plein, Reparation: -1,
	}
	if jr.Veterans > 0 {
		jr.Veterans--
		a.Veteran = true
	}
	e.Avions = append(e.Avions, a)
	return a
}

// AvionsDe : avions encore en service d'un joueur.
func (e *Etat) AvionsDe(j int) []*Avion {
	var l []*Avion
	for _, a := range e.Avions {
		if a.Joueur == j && a.Actif() {
			l = append(l, a)
		}
	}
	return l
}

func (e *Etat) PistesDe(j int) []*Piste {
	var l []*Piste
	for _, p := range e.Pistes {
		if p.Joueur == j {
			l = append(l, p)
		}
	}
	return l
}

// PistesSur : pistes posées sur une case (plusieurs sur une île).
func (e *Etat) PistesSur(k Case) []*Piste {
	var l []*Piste
	for _, p := range e.Pistes {
		if p.Case == k {
			l = append(l, p)
		}
	}
	return l
}

func (e *Etat) PisteDe(j int, k Case) *Piste {
	for _, p := range e.Pistes {
		if p.Case == k && p.Joueur == j {
			return p
		}
	}
	return nil
}

// PionsLibres : pions d'avions d'un type encore disponibles pour un joueur.
func (e *Etat) PionsLibres(j int, t TypeAvion) int {
	n := t.Car().Pions
	for _, a := range e.AvionsDe(j) {
		if a.Type == t {
			n--
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Phase d'équipement

func (e *Etat) PeutAcheter(j int, t TypeAvion) error {
	switch {
	case e.Phase != PhaseEquipement || e.Courant != j:
		return fmt.Errorf("les achats se font pendant la phase d'équipement")
	case e.Joueurs[j].Capital < t.Car().Prix:
		return fmt.Errorf("un %s coûte %s", t, Francs(t.Car().Prix))
	case e.PionsLibres(j, t) <= 0:
		return fmt.Errorf("plus de pion %s disponible", t)
	}
	return nil
}

// Acheter : l'avion neuf attend à Paris, le plein fait.
func (e *Etat) Acheter(j int, t TypeAvion) (*Avion, error) {
	if err := e.PeutAcheter(j, t); err != nil {
		return nil, err
	}
	e.Joueurs[j].Capital -= t.Car().Prix
	a := e.nouvelAvion(j, t)
	e.noter(a, "achat à Paris")
	v := ""
	if a.Veteran {
		v = ", piloté par un vétéran"
	}
	e.Log("%s achète un %s (%s)%s.", e.Nom(j), t, Francs(t.Car().Prix), v)
	return a, nil
}

// PeutConstruire : la case doit être une terre libre, sans ville ni piste
// adverse ; sur une île, plusieurs joueurs peuvent avoir chacun leur piste.
func (e *Etat) PeutConstruire(j int, k Case) error {
	switch {
	case e.Phase != PhaseEquipement || e.Courant != j:
		return fmt.Errorf("on construit pendant la phase d'équipement")
	case e.tentee:
		return fmt.Errorf("une seule tentative de construction par tour")
	case !k.Valide() || e.P.Terrain(k) == Mer:
		return fmt.Errorf("impossible de construire en pleine mer")
	case e.P.Ville(k) >= 0:
		return fmt.Errorf("la case doit être libre : pas de piste sur une ville")
	case len(e.PistesDe(j)) >= PionsPistes:
		return fmt.Errorf("vos %d pions d'aérodrome sont posés", PionsPistes)
	}
	ps := e.PistesSur(k)
	if len(ps) > 0 && (e.P.Terrain(k) != Ile || e.PisteDe(j, k) != nil) {
		return fmt.Errorf("la case est déjà occupée par une piste")
	}
	if e.Joueurs[j].Capital < e.P.Terrain(k).Valeur()*1000 {
		return fmt.Errorf("pas assez d'argent, même avec un 1 au dé")
	}
	return nil
}

// DevisPiste : jet du dé de construction ; le prix est annoncé, le joueur
// décide ensuite (Construire) de bâtir ou non.
func (e *Etat) DevisPiste(j int, k Case) (int, error) {
	if err := e.PeutConstruire(j, k); err != nil {
		return 0, err
	}
	e.tentee = true
	d := e.De()
	prix := d * e.P.Terrain(k).Valeur() * 1000
	e.evenement(j, k, false, []int{d}, "%s étudie une piste en %s : dé %d, devis %s.", e.Nom(j), e.P.Terrain(k), d, Francs(prix))
	return prix, nil
}

func (e *Etat) Construire(j int, k Case, prix int) error {
	if e.Joueurs[j].Capital < prix {
		return fmt.Errorf("le devis (%s) dépasse votre capital", Francs(prix))
	}
	e.Joueurs[j].Capital -= prix
	e.Pistes = append(e.Pistes, &Piste{Joueur: j, Case: k})
	e.Log("%s construit une piste privée (%s).", e.Nom(j), Francs(prix))
	return nil
}

// ---------------------------------------------------------------------------
// Atterrissage, courrier, ravitaillement

// PeutAtterrir : villes, pistes privées (de tous les joueurs) et, pour
// l'hydravion, les cases de fleuve.
func (e *Etat) PeutAtterrir(a *Avion, k Case) bool {
	return e.P.Ville(k) >= 0 || len(e.PistesSur(k)) > 0 || (a.Type == Hydravion && e.P.Fleuve(k))
}

// Ravitaillable : ville ou piste privée.
func (e *Etat) Ravitaillable(k Case) bool {
	return e.P.Ville(k) >= 0 || len(e.PistesSur(k)) > 0
}

// VilleDuCourrier : ville où un avion posé en k prend un sac — la ville
// elle-même, ou depuis une piste privée, la ville voisine la plus chère.
// Paris ne fournit pas de courrier.
func (e *Etat) VilleDuCourrier(a *Avion, k Case) int {
	if v := e.P.Ville(k); v >= 0 {
		if v == e.P.Paris || slices.Contains(a.PrisCeTour, v) {
			return -1
		}
		return v
	}
	if len(e.PistesSur(k)) == 0 {
		return -1
	}
	best := -1
	for _, n := range k.Voisins() {
		v := e.P.Ville(n)
		if v < 0 || v == e.P.Paris || slices.Contains(a.PrisCeTour, v) {
			continue
		}
		if best < 0 || e.P.Villes[v].Valeur > e.P.Villes[best].Valeur {
			best = v
		}
	}
	return best
}

// atterrir : l'avion se pose, prend le courrier ou le livre à Paris.
func (e *Etat) atterrir(a *Avion) {
	a.Etat = AuSol
	a.Sursis = false
	k := a.Case
	j := a.Joueur
	nom := e.NomCase(k)
	if e.P.Ville(k) == e.P.Paris {
		e.livrer(a)
		return
	}
	if v := e.VilleDuCourrier(a, k); v >= 0 {
		a.Sacs = append(a.Sacs, Sac{v, e.Tour})
		a.PrisCeTour = append(a.PrisCeTour, v)
		vn := e.P.Villes[v].Nom
		if e.P.Ville(k) < 0 {
			vn += " (par la piste voisine)"
		}
		e.noter(a, "%s (1 sac)", vn)
		e.evenement(j, k, false, nil, "%s : le %s se pose à %s et prend un sac de courrier.", e.Nom(j), a.Nom(), vn)
		return
	}
	e.noter(a, "escale %s", nom)
	e.Log("%s : le %s se pose (%s).", e.Nom(j), a.Nom(), nom)
}

// livrer : les sacs rapportés à Paris sont payés 1 000 F × valeur de la ville ;
// la prime de première liaison est réglée à la fin du tour.
func (e *Etat) livrer(a *Avion) {
	j := a.Joueur
	if len(a.Sacs) == 0 {
		e.noter(a, "Paris")
		e.Log("%s : le %s rentre à Paris sans courrier.", e.Nom(j), a.Nom())
		return
	}
	gain := 0
	for _, s := range a.Sacs {
		gain += e.P.Villes[s.Ville].Valeur * 1000
		e.ordreRemise++
		e.Remises = append(e.Remises, Remise{j, s.Ville, s.Tour, e.ordreRemise})
	}
	jr := &e.Joueurs[j]
	jr.Capital += gain
	jr.Gains += gain
	jr.Rapportes += len(a.Sacs)
	e.noter(a, "Paris : %d sac(s), %s", len(a.Sacs), Francs(gain))
	e.evenement(j, a.Case, false, nil, "%s : le %s livre %d sac(s) à Paris (%s). Total : %d sac(s).", e.Nom(j), a.Nom(), len(a.Sacs), Francs(gain), jr.Rapportes)
	a.Sacs = nil
}

// primes : à la fin du tour, chaque ville reliée à Paris pour la première
// fois double la valeur d'un sac ; si plusieurs avions en rapportent, le
// courrier le plus récent (pris au tour le plus élevé) l'emporte.
func (e *Etat) primes() {
	for v := range e.P.Villes {
		if e.Reliees[v] {
			continue
		}
		best := -1
		for i, r := range e.Remises {
			if r.Ville != v {
				continue
			}
			if best < 0 || r.TourPris > e.Remises[best].TourPris {
				best = i
			}
		}
		if best < 0 {
			continue
		}
		r := e.Remises[best]
		e.Reliees[v] = true
		prime := e.P.Villes[v].Valeur * 1000
		e.Joueurs[r.Joueur].Capital += prime
		e.Joueurs[r.Joueur].Gains += prime
		e.evenement(r.Joueur, e.P.Villes[v].Case, false, nil, "Première liaison %s – Paris : prime de %s pour %s.", e.P.Villes[v].Nom, Francs(prime), e.Nom(r.Joueur))
	}
	e.Remises = e.Remises[:0]
}

// PrixPlein : prix du plein d'un avion sur sa case (gratuit sur ses pistes,
// double sur une piste adverse, -1 si l'on ne peut pas se ravitailler).
func (e *Etat) PrixPlein(a *Avion) (prix int, proprio int) {
	k := a.Case
	base := a.Type.Car().PrixPlein
	if e.P.Ville(k) >= 0 {
		return base, -1
	}
	ps := e.PistesSur(k)
	if len(ps) == 0 {
		return -1, -1
	}
	if e.PisteDe(a.Joueur, k) != nil {
		return 0, -1
	}
	return 2 * base, ps[0].Joueur
}

// ravitailler : phase 6, chaque avion posé sur une ville ou une piste fait le plein.
func (e *Etat) ravitailler(j int) {
	for _, a := range e.AvionsDe(j) {
		if a.Etat != AuSol || a.Carburant >= 2*a.Type.Car().Plein {
			continue
		}
		prix, proprio := e.PrixPlein(a)
		if prix < 0 {
			continue
		}
		jr := &e.Joueurs[j]
		if jr.Capital < prix {
			e.Log("%s n'a pas de quoi payer le plein du %s (%s).", e.Nom(j), a.Nom(), Francs(prix))
			continue
		}
		jr.Capital -= prix
		if proprio >= 0 {
			e.Joueurs[proprio].Capital += prix
			e.Log("%s : plein du %s sur la piste de %s, %s.", e.Nom(j), a.Nom(), e.Nom(proprio), Francs(prix))
		} else if prix > 0 {
			e.Log("%s : plein du %s, %s.", e.Nom(j), a.Nom(), Francs(prix))
		}
		a.Carburant = 2 * a.Type.Car().Plein
		if prix > 0 {
			e.noter(a, "plein %s", Francs(prix))
		} else {
			e.noter(a, "plein gratuit")
		}
	}
}

// Entreposer : sur sa propre piste, l'avion y laisse ses sacs.
func (e *Etat) Entreposer(a *Avion) error {
	p := e.PisteDe(a.Joueur, a.Case)
	if p == nil || a.Etat != AuSol || len(a.Sacs) == 0 {
		return fmt.Errorf("rien à entreposer ici")
	}
	p.Stock = append(p.Stock, a.Sacs...)
	e.noter(a, "%d sac(s) entreposé(s)", len(a.Sacs))
	e.Log("%s entrepose %d sac(s) sur sa piste.", e.Nom(a.Joueur), len(a.Sacs))
	a.Sacs = nil
	return nil
}

// Reprendre : l'avion charge le courrier entreposé sur sa piste.
func (e *Etat) Reprendre(a *Avion) error {
	p := e.PisteDe(a.Joueur, a.Case)
	if p == nil || a.Etat != AuSol || len(p.Stock) == 0 {
		return fmt.Errorf("pas de courrier entreposé ici")
	}
	a.Sacs = append(a.Sacs, p.Stock...)
	e.noter(a, "%d sac(s) repris", len(p.Stock))
	e.Log("%s reprend %d sac(s) entreposé(s).", e.Nom(a.Joueur), len(p.Stock))
	p.Stock = nil
	return nil
}

// NomCase : ville, piste ou terrain.
func (e *Etat) NomCase(k Case) string {
	if v := e.P.Ville(k); v >= 0 {
		return e.P.Villes[v].Nom
	}
	if ps := e.PistesSur(k); len(ps) > 0 {
		return "piste de " + e.Nom(ps[0].Joueur)
	}
	t := e.P.Terrain(k).String()
	if e.P.Fleuve(k) {
		t += ", fleuve"
	}
	return t
}

// Francs : 12 000 F.
func Francs(n int) string {
	s := fmt.Sprint(abs(n))
	var b []byte
	for i := range len(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ' ')
		}
		b = append(b, s[i])
	}
	if n < 0 {
		return "-" + string(b) + " F"
	}
	return string(b) + " F"
}
