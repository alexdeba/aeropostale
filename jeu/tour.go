package jeu

// Le tour de jeu : marqueur de tour, météo, puis pour chaque joueur
// équipement, déplacement (prise du courrier à chaque escale) et ravitaillement.

func (e *Etat) Partie() {
	for !e.Fini {
		e.JouerTour()
	}
}

func (e *Etat) JouerTour() {
	e.Tour++
	e.Phase = PhaseMeteo
	e.Vent = e.De()
	e.evenement(-1, Nulle, false, []int{e.Vent}, "Tour %d : le vent souffle vers le %s (%d).", e.Tour, NomDirection(e.Vent), e.Vent)
	e.Pause(false)
	for _, j := range e.Ordre {
		e.JouerJoueur(j)
		if e.Fini {
			return
		}
	}
	e.primes()
	e.verifierFin()
}

// JouerJoueur : les quatre phases d'un joueur.
func (e *Etat) JouerJoueur(j int) {
	e.Courant = j
	e.tentee = false
	for _, a := range e.AvionsDe(j) {
		a.PrisCeTour = a.PrisCeTour[:0]
	}
	e.majBloque(j)
	if e.Joueurs[j].Bloque {
		e.Log("%s n'a plus d'avion ni de quoi en acheter.", e.Nom(j))
		return
	}
	e.Phase = PhaseEquipement
	e.Ctrl[j].Equiper(e, j)

	e.Phase = PhaseDeplacement
	joues := map[int]bool{}
	for {
		var restants []*Avion
		for _, a := range e.AvionsDe(j) {
			if joues[a.ID] || a.Etat != AuSol {
				continue
			}
			if a.Reparation >= e.Tour {
				joues[a.ID] = true
				e.noter(a, "en réparations")
				e.Log("%s : le %s est en réparations.", e.Nom(j), a.Nom())
				continue
			}
			restants = append(restants, a)
		}
		if len(restants) == 0 {
			break
		}
		a := e.Ctrl[j].ChoisirAvion(e, j, restants)
		if a == nil || !contientAvion(restants, a) {
			break
		}
		joues[a.ID] = true
		e.deplacer(a)
	}

	e.Phase = PhaseRavitaillement
	e.ravitailler(j)
	e.Pause(false)

	if e.Joueurs[j].Rapportes >= SacsVictoire {
		e.Fini = true
		e.Gagnant = j
		e.Phase = PhaseFin
		e.evenement(j, e.P.CaseParis(), false, nil, "%s a rapporté %d sacs de courrier à Paris et gagne la partie !", e.Nom(j), e.Joueurs[j].Rapportes)
	}
}

func contientAvion(l []*Avion, a *Avion) bool {
	for _, x := range l {
		if x == a {
			return true
		}
	}
	return false
}

// majBloque : sans avion et sans de quoi en acheter, un joueur ne peut plus rien faire.
func (e *Etat) majBloque(j int) {
	jr := &e.Joueurs[j]
	jr.Bloque = len(e.AvionsDe(j)) == 0 && jr.Capital < Types[MoyenCourrier].Prix
}

// verifierFin : tous bloqués, ou garde-fou du nombre de tours : le plus de
// sacs rapportés l'emporte (puis le capital).
func (e *Etat) verifierFin() {
	tous := true
	for j := range e.Joueurs {
		e.majBloque(j)
		tous = tous && e.Joueurs[j].Bloque
	}
	if !tous && e.Tour < ToursMax {
		return
	}
	best := 0
	for j, jr := range e.Joueurs {
		b := e.Joueurs[best]
		if jr.Rapportes > b.Rapportes || (jr.Rapportes == b.Rapportes && jr.Capital > b.Capital) {
			best = j
		}
	}
	e.Fini = true
	e.Gagnant = best
	e.Phase = PhaseFin
	e.evenement(best, Nulle, false, nil, "Plus personne ne vole : %s l'emporte avec %d sac(s) rapporté(s).", e.Nom(best), e.Joueurs[best].Rapportes)
}
