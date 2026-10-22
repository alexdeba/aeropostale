package jeu

import "fmt"

// Le déplacement d'un avion (tableau 2 de la règle) : jet d'autonomie,
// décollage, vol de case en case, pannes, crash, jet de survie.

// Ordre de vol donné par un contrôleur : entrer dans une case voisine ou se poser.
type Ordre struct {
	Atterrir bool
	Vers     Case
}

// Cout d'entrée dans une case voisine, en demi-points : ½ point vent arrière,
// 2 vent debout, 1 vent de côté ; le double sur une montagne.
func (e *Etat) Cout(de, vers Case) int {
	d := de.DirectionVers(vers)
	c := 2
	switch d {
	case e.Vent:
		c = 1
	case Opposee(e.Vent):
		c = 4
	}
	if e.P.Terrain(vers) == Montagne {
		c *= 2
	}
	return c
}

// Portee : coût minimal (demi-points) pour atteindre chaque case depuis k
// sans dépasser budget ; -1 hors de portée. prec donne le trajet.
func (e *Etat) Portee(k Case, budget int) (cout [NbRang][NbCol]int, prec [NbRang][NbCol]Case) {
	for r := range NbRang {
		for c := range NbCol {
			cout[r][c] = -1
		}
	}
	if budget < 0 {
		return
	}
	cout[k.R][k.C] = 0
	// Dijkstra à petits coûts entiers : files par coût
	files := make([][]Case, budget+1)
	files[0] = append(files[0], k)
	for b := 0; b <= budget; b++ {
		for i := 0; i < len(files[b]); i++ {
			x := files[b][i]
			if cout[x.R][x.C] != b {
				continue
			}
			for _, n := range x.Voisins() {
				nb := b + e.Cout(x, n)
				if nb > budget {
					continue
				}
				if c := cout[n.R][n.C]; c < 0 || nb < c {
					cout[n.R][n.C] = nb
					prec[n.R][n.C] = x
					files[nb] = append(files[nb], n)
				}
			}
		}
	}
	return
}

// Chemin de k vers dest d'après prec (sans k).
func Chemin(prec [NbRang][NbCol]Case, k, dest Case) []Case {
	var ch []Case
	for x := dest; x != k; x = prec[x.R][x.C] {
		ch = append(ch, x)
	}
	for i, j := 0, len(ch)-1; i < j; i, j = i+1, j-1 {
		ch[i], ch[j] = ch[j], ch[i]
	}
	return ch
}

// AutonomieRestante : demi-points avant la zone de panne (négatif si dépassée).
func (a *Avion) AutonomieRestante() int { return a.Jet - a.Conso }

// PeutSePoser : l'avion en vol peut atterrir sur sa case — pas sur celle
// d'où il vient de décoller sans avoir bougé (ce serait relancer l'autonomie
// pour un point).
func (e *Etat) PeutSePoser(a *Avion) bool {
	return a.Etat == EnVol && e.PeutAtterrir(a, a.Case) && (len(a.Trajet) > 1 || a.Case != a.Depart)
}

// CoupsVol : cases voisines où l'avion en vol peut entrer.
func (e *Etat) CoupsVol(a *Avion) []Case {
	var l []Case
	if a.Etat != EnVol {
		return nil
	}
	for _, n := range a.Case.Voisins() {
		if e.Cout(a.Case, n) <= a.Carburant {
			l = append(l, n)
		}
	}
	return l
}

// JetAutonomie : un dé pour le moyen-courrier, deux pour les autres, +1 au vétéran.
func (e *Etat) jetAutonomie(a *Avion) {
	n := a.Type.Car().DesAutonomie
	des := make([]int, n)
	tot := 0
	for i := range des {
		des[i] = e.De()
		tot += des[i]
	}
	v := ""
	if a.Veteran {
		tot++
		v = " +1 (vétéran)"
	}
	a.Jet, a.Conso = 2*tot, 0
	e.evenement(a.Joueur, a.Case, false, des, "%s : jet d'autonomie du %s : %s%s = %d point(s) en toute sécurité.", e.Nom(a.Joueur), a.Nom(), desTexte(des), v, tot)
}

func desTexte(d []int) string {
	s := ""
	for i, x := range d {
		if i > 0 {
			s += " + "
		}
		s += fmt.Sprint(x)
	}
	return s
}

// deplacer : phase de déplacement d'un avion, jusqu'à ce qu'il reste au sol.
func (e *Etat) deplacer(a *Avion) {
	e.Actif = a.ID
	defer func() { e.Actif = -1 }()
	j := a.Joueur
	for a.Actif() && a.Etat == AuSol {
		if a.Carburant < 2 {
			e.Log("%s : le %s n'a plus de quoi décoller.", e.Nom(j), a.Nom())
			return
		}
		e.jetAutonomie(a)
		e.Pause(false)
		if !e.Ctrl[j].Decoller(e, a) {
			a.Jet, a.Conso = 0, 0
			e.Log("%s : le %s reste au sol (%s).", e.Nom(j), a.Nom(), e.NomCase(a.Case))
			return
		}
		a.Carburant -= 2
		a.Conso = 2
		a.Etat = EnVol
		a.Depart = a.Case
		a.Trajet = []Case{a.Case}
		e.Log("%s : le %s décolle de %s.", e.Nom(j), a.Nom(), e.NomCase(a.Case))
		if !e.voler(a) {
			return
		}
		a.Jet, a.Conso = 0, 0
		e.Pause(false)
		if e.PisteDe(j, a.Case) != nil {
			e.Ctrl[j].Escale(e, a)
		}
	}
}

// voler : rend vrai si l'avion s'est posé normalement (il peut redécoller),
// faux si son tour est fini (crash, atterrissage forcé).
func (e *Etat) voler(a *Avion) bool {
	j := a.Joueur
	erreurs := 0
	for {
		coups := e.CoupsVol(a)
		posable := e.PeutSePoser(a)
		if len(coups) == 0 {
			if e.PeutAtterrir(a, a.Case) {
				e.atterrir(a)
				return true
			}
			e.evenement(j, a.Case, true, nil, "%s : panne sèche ! Le %s n'a plus de carburant en plein vol.", e.Nom(j), a.Nom())
			e.crash(a, a.Case)
			return false
		}
		o := e.Ctrl[j].Voler(e, a)
		if erreurs > 20 { // contrôleur défaillant : on se pose ou on prend le premier coup
			o, erreurs = Ordre{Atterrir: posable, Vers: coups[0]}, 0
		}
		if o.Atterrir {
			if posable {
				e.atterrir(a)
				return true
			}
			erreurs++
		} else if !contient(coups, o.Vers) {
			erreurs++
		} else {
			if a.Conso >= a.Jet && !e.jetPanne(a) {
				return false
			}
			c := e.Cout(a.Case, o.Vers)
			a.Carburant -= c
			a.Conso += c
			a.Case = o.Vers
			a.Trajet = append(a.Trajet, o.Vers)
			a.Sursis = a.Conso >= a.Jet
			e.Pause(true)
		}
	}
}

func contient(l []Case, k Case) bool {
	for _, x := range l {
		if x == k {
			return true
		}
	}
	return false
}

// jetPanne : l'autonomie est épuisée en plein vol ; vrai si le vol continue.
func (e *Etat) jetPanne(a *Avion) bool {
	j := a.Joueur
	d := e.De()
	switch d {
	case 1:
		e.evenement(j, a.Case, true, []int{d}, "%s : jet de panne %d, crash mortel ! Le %s et son pilote sont perdus.", e.Nom(j), d, a.Nom())
		e.noter(a, "crash mortel (panne 1)")
		e.perdre(a, false)
		return false
	case 2:
		e.evenement(j, a.Case, true, []int{d}, "%s : jet de panne %d, le %s s'écrase sur place.", e.Nom(j), d, a.Nom())
		e.crash(a, a.Case)
		return false
	case 3:
		e.evenement(j, a.Case, true, []int{d}, "%s : jet de panne %d, le %s tente de se poser sur une case voisine.", e.Nom(j), d, a.Nom())
		choix := a.Case.Voisins()
		k := e.Ctrl[j].CaseDeCrash(e, a, choix)
		if !contient(choix, k) {
			k = choix[0]
		}
		e.crash(a, k)
		return false
	case 4:
		a.Etat = AuSol
		a.Sursis = false
		a.Reparation = e.Tour + 1
		e.noter(a, "(panne 4)")
		e.evenement(j, a.Case, true, []int{d}, "%s : jet de panne %d, le %s se pose sans casse mais passe un tour en réparations.", e.Nom(j), d, a.Nom())
		return false
	}
	e.evenement(j, a.Case, false, []int{d}, "%s : jet de panne %d, pas de panne, le %s continue (un jet par case).", e.Nom(j), d, a.Nom())
	e.Pause(false)
	return true
}

// crash : l'avion se pose en catastrophe sur k. L'hydravion sur un fleuve
// reste intact sur 3 à 6 ; le pilote survit si son dé atteint la valeur du terrain.
func (e *Etat) crash(a *Avion, k Case) {
	j := a.Joueur
	a.Case = k
	a.Etat = AuSol
	a.Sursis = false
	intact := false
	var des []int
	txt := ""
	if a.Type == Hydravion && e.P.Fleuve(k) {
		d := e.De()
		des = append(des, d)
		intact = d >= 3
		if intact {
			txt = fmt.Sprintf("l'hydravion amerrit sur le fleuve (%d), intact ; ", d)
		} else {
			txt = fmt.Sprintf("l'hydravion se brise sur le fleuve (%d) ; ", d)
		}
	}
	d := e.De()
	des = append(des, d)
	t := e.P.Terrain(k)
	survit := d >= t.Valeur()
	switch {
	case survit && intact:
		a.Carburant = 2 * a.Type.Car().Plein
		e.noter(a, "crash (%s), avion et pilote intacts", t)
		e.evenement(j, k, true, des, "%s : %sjet de survie %d (%s, %d requis) : le pilote s'en sort, il redécollera au prochain tour.", e.Nom(j), txt, d, t, t.Valeur())
	case survit:
		e.noter(a, "crash (%s), pilote sauvé", t)
		e.evenement(j, k, true, des, "%s : %sjet de survie %d (%s, %d requis) : le pilote survit et rentre à Paris en vétéran ; l'avion est perdu.", e.Nom(j), txt, d, t, t.Valeur())
		e.perdre(a, true)
	default:
		e.noter(a, "crash (%s), pilote perdu", t)
		e.evenement(j, k, true, des, "%s : %sjet de survie %d (%s, %d requis) : le pilote ne survit pas, l'avion est perdu.", e.Nom(j), txt, d, t, t.Valeur())
		e.perdre(a, false)
	}
}

// perdre : l'avion et son courrier sont perdus ; le pilote rescapé devient vétéran.
func (e *Etat) perdre(a *Avion, pilote bool) {
	a.Etat = Detruit
	a.Sacs = nil
	jr := &e.Joueurs[a.Joueur]
	if pilote {
		jr.Veterans++
	} else {
		jr.Pilotes++
	}
}
