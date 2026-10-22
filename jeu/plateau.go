// Package jeu : les règles d'Aéropostale (François Nédélec, Jeux & Stratégie
// 1983), sans interface.
package jeu

// Le plateau : 27 rangées d'hexagones pointe en haut. Les rangées paires sont
// décalées d'un demi-pas vers la droite et comptent 28 cases, les impaires 29.
const (
	NbRang = 27
	NbCol  = 29
)

func LargeurRangee(r int) int { return NbCol - 1 + r%2 }

type Case struct{ C, R int }

var Nulle = Case{-1, -1}

func (k Case) Valide() bool {
	return k.R >= 0 && k.R < NbRang && k.C >= 0 && k.C < LargeurRangee(k.R)
}

type Terrain uint8

const (
	Mer Terrain = iota
	Ile
	Plaine
	Desert
	Jungle
	Montagne
)

var nomsTerrain = [...]string{"mer", "île", "plaine", "désert", "jungle", "montagne"}

func (t Terrain) String() string { return nomsTerrain[t] }

// Valeur : coefficient du terrain pour le prix d'une piste et le jet de
// survie (règle : plaine 2, désert 3, jungle 4, montagne 5, île 6). La mer
// n'a pas de valeur dans la règle : on lui donne celle de l'île.
func (t Terrain) Valeur() int {
	switch t {
	case Plaine:
		return 2
	case Desert:
		return 3
	case Jungle:
		return 4
	case Montagne:
		return 5
	}
	return 6
}

type Ville struct {
	Nom    string
	Valeur int // valeur d'un sac, en milliers de francs (0 pour Paris)
	Case   Case
}

// Directions de la rose des vents imprimée sur le plateau.
const (
	NordOuest = 1 + iota
	Ouest
	SudOuest
	SudEst
	Est
	NordEst
)

var nomsDirection = [...]string{"", "nord-ouest", "ouest", "sud-ouest", "sud-est", "est", "nord-est"}

func NomDirection(d int) string { return nomsDirection[d] }

func Opposee(d int) int { return (d+2)%6 + 1 }

// Voisin dans la direction d (1 à 6), éventuellement hors plateau.
func (k Case) Voisin(d int) Case {
	dec := 0
	if k.R%2 == 0 {
		dec = 1
	}
	switch d {
	case NordOuest:
		return Case{k.C - 1 + dec, k.R - 1}
	case NordEst:
		return Case{k.C + dec, k.R - 1}
	case Ouest:
		return Case{k.C - 1, k.R}
	case Est:
		return Case{k.C + 1, k.R}
	case SudOuest:
		return Case{k.C - 1 + dec, k.R + 1}
	case SudEst:
		return Case{k.C + dec, k.R + 1}
	}
	return Nulle
}

// Voisins sur le plateau.
func (k Case) Voisins() []Case {
	v := make([]Case, 0, 6)
	for d := 1; d <= 6; d++ {
		if n := k.Voisin(d); n.Valide() {
			v = append(v, n)
		}
	}
	return v
}

// DirectionVers : direction d'une case voisine (0 si elle ne l'est pas).
func (k Case) DirectionVers(n Case) int {
	for d := 1; d <= 6; d++ {
		if k.Voisin(d) == n {
			return d
		}
	}
	return 0
}

func (k Case) cube() (int, int, int) {
	q := k.C - (k.R+(k.R&1))/2
	return q, k.R, -q - k.R
}

func Distance(a, b Case) int {
	x1, y1, z1 := a.cube()
	x2, y2, z2 := b.cube()
	return max(abs(x1-x2), abs(y1-y2), abs(z1-z2))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Plateau : terrain, fleuves et villes, relevés sur l'encart.
type Plateau struct {
	terrain [NbRang][NbCol]Terrain
	fleuve  [NbRang][NbCol]bool
	Villes  []Ville
	ville   map[Case]int
	Paris   int // indice de Paris dans Villes
}

func NouveauPlateau() *Plateau {
	p := &Plateau{ville: map[Case]int{}}
	for r, s := range rangeesPlateau {
		for c := range len(s) {
			l := s[c]
			if l >= 'A' && l <= 'Z' {
				p.fleuve[r][c] = true
				l += 'a' - 'A'
			}
			p.terrain[r][c] = map[byte]Terrain{'m': Mer, 'i': Ile, 'p': Plaine, 'd': Desert, 'j': Jungle, 'n': Montagne}[l]
		}
	}
	p.Villes = villesPlateau
	for i, v := range p.Villes {
		p.ville[v.Case] = i
		if v.Valeur == 0 {
			p.Paris = i
		}
	}
	return p
}

func (p *Plateau) Terrain(k Case) Terrain { return p.terrain[k.R][k.C] }
func (p *Plateau) Fleuve(k Case) bool     { return p.fleuve[k.R][k.C] }

// Ville sur la case (indice dans Villes, -1 sinon).
func (p *Plateau) Ville(k Case) int {
	if i, ok := p.ville[k]; ok {
		return i
	}
	return -1
}

func (p *Plateau) CaseParis() Case { return p.Villes[p.Paris].Case }

// Cases : toutes les cases du plateau, rangée par rangée.
func Cases() []Case {
	var l []Case
	for r := range NbRang {
		for c := range LargeurRangee(r) {
			l = append(l, Case{c, r})
		}
	}
	return l
}
