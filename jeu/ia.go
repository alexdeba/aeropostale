package jeu

import (
	"math"
	"slices"
)

// IA : un joueur simple mais prudent. Chaque avion suit une mission (aller
// vers une ville cible en ramassant le courrier des escales, puis rentrer à
// Paris). À chaque escale, après le jet d'autonomie, il choisit la meilleure
// escale accessible sans entrer en zone de panne, ou reste au sol.
type IA struct {
	Audace    float64 // goût des longs voyages et des dépenses (0 à 1)
	Temeraire bool    // vole au-delà de son autonomie (essais, captures des pannes)
	missions  map[int]*mission
	plans     map[int][]Case
}

type mission struct {
	cible  int // ville visée (indice), Paris au retour
	retour bool
}

func NouvelleIA(audace float64) *IA {
	return &IA{Audace: audace, missions: map[int]*mission{}, plans: map[int][]Case{}}
}

func (ia *IA) init() {
	if ia.missions == nil {
		ia.missions = map[int]*mission{}
		ia.plans = map[int][]Case{}
	}
}

// ---------------------------------------------------------------------------
// Réseau des escales : villes et pistes, reliées quand un saut est raisonnable
// pour le type d'avion.

func sautMax(t TypeAvion) int {
	switch t {
	case MoyenCourrier:
		return 4
	case Hydravion:
		return 7
	}
	return 8
}

func (e *Etat) escales() []Case {
	var l []Case
	for _, v := range e.P.Villes {
		l = append(l, v.Case)
	}
	for _, p := range e.Pistes {
		if !slices.Contains(l, p.Case) {
			l = append(l, p.Case)
		}
	}
	return l
}

// distancesReseau : distance (cases + 1,5 par escale) de chaque escale à but.
func (e *Etat) distancesReseau(t TypeAvion, but Case, extra Case) map[Case]float64 {
	nds := e.escales()
	if extra.Valide() && !slices.Contains(nds, extra) {
		nds = append(nds, extra)
	}
	dist := map[Case]float64{}
	for _, n := range nds {
		dist[n] = math.Inf(1)
	}
	dist[but] = 0
	fait := map[Case]bool{}
	m := sautMax(t)
	for {
		var u Case
		best := math.Inf(1)
		for _, n := range nds {
			if !fait[n] && dist[n] < best {
				u, best = n, dist[n]
			}
		}
		if math.IsInf(best, 1) {
			break
		}
		fait[u] = true
		for _, n := range nds {
			d := Distance(u, n)
			if fait[n] || d > m {
				continue
			}
			if nd := best + float64(d) + 1.5; nd < dist[n] {
				dist[n] = nd
			}
		}
	}
	return dist
}

// ---------------------------------------------------------------------------
// Missions

func (ia *IA) mission(e *Etat, a *Avion) *mission {
	ia.init()
	m := ia.missions[a.ID]
	paris := e.P.Paris
	reste := SacsVictoire - e.Joueurs[a.Joueur].Rapportes
	seuil := 3
	if a.Type != MoyenCourrier {
		seuil = 4
	}
	switch {
	case m == nil:
		m = &mission{cible: -1}
		ia.missions[a.ID] = m
	case !m.retour && (len(a.Sacs) >= seuil || len(a.Sacs) >= reste || e.P.Ville(a.Case) == m.cible):
		m.retour = true
	case m.retour && e.P.Ville(a.Case) == paris && len(a.Sacs) == 0:
		m.retour, m.cible = false, -1
	}
	if m.retour {
		m.cible = paris
	}
	if m.cible < 0 {
		m.cible = ia.choisirCible(e, a)
	}
	return m
}

// choisirCible : ville joignable, chère et pas trop lointaine ; les villes
// jamais reliées à Paris attirent davantage.
func (ia *IA) choisirCible(e *Etat, a *Avion) int {
	dp := e.distancesReseau(a.Type, e.P.CaseParis(), a.Case)
	best, bs := e.P.Paris, -1.0
	for v, vi := range e.P.Villes {
		if v == e.P.Paris {
			continue
		}
		d := dp[vi.Case]
		if math.IsInf(d, 1) {
			continue
		}
		val := float64(vi.Valeur)
		if !e.Reliees[v] {
			val *= 1.6
		}
		// les autres avions du joueur partent ailleurs
		for id, m := range ia.missions {
			if id != a.ID && m.cible == v && e.Avions[id].Actif() {
				val *= 0.5
			}
		}
		s := (6 + val*(0.4+ia.Audace)) / (6 + d*(1.3-ia.Audace*0.6))
		s *= 1 + 0.15*float64((v*7+a.ID*3)%5)/4 // un peu de variété
		if s > bs {
			best, bs = v, s
		}
	}
	return best
}

// ---------------------------------------------------------------------------
// Choix de l'escale

type escale struct {
	k      Case
	score  float64
	chemin []Case
}

// meilleureEscale : depuis la position de l'avion, avec budget demi-points
// (sans le décollage), l'escale la plus intéressante.
func (ia *IA) meilleureEscale(e *Etat, a *Avion, budget int) (escale, bool) {
	m := ia.mission(e, a)
	but := e.P.Villes[m.cible].Case
	dr := e.distancesReseau(a.Type, but, a.Case)
	cout, prec := e.Portee(a.Case, budget)
	ici := dr[a.Case]
	if math.IsInf(ici, 1) {
		ici = float64(Distance(a.Case, but)) * 1.5
	}
	best := escale{score: math.Inf(-1)}
	trouve := false
	for _, k := range Cases() {
		c := cout[k.R][k.C]
		if c <= 0 || !e.PeutAtterrir(a, k) {
			continue
		}
		s := 0.0
		poids := 0.6                           // valeur d'une case gagnée vers le but
		charge := m.retour && len(a.Sacs) >= 6 // trop de courrier en jeu : on rentre
		if v := e.VilleDuCourrier(a, k); v >= 0 {
			b := 3 + float64(e.P.Villes[v].Valeur)/4
			if charge {
				b *= 0.3
			}
			s += b
		}
		if charge {
			poids = 1.4
		}
		if e.P.Ville(k) == e.P.Paris && len(a.Sacs) > 0 {
			s += 4 * float64(len(a.Sacs))
			if e.Joueurs[a.Joueur].Rapportes+len(a.Sacs) >= SacsVictoire {
				s += 100
			}
		}
		d := dr[k]
		if math.IsInf(d, 1) {
			d = float64(Distance(k, but)) * 1.5
		}
		s += (ici - d) * poids
		if prix, proprio := e.prixPleinEn(a, k); proprio >= 0 && proprio != a.Joueur {
			s -= float64(prix) / 1500
		} else if prix < 0 { // fleuve : ni courrier ni plein
			s -= 1.5
		}
		s -= float64(c) * 0.02
		if ia.Temeraire {
			s += float64(c) * 0.5
		}
		if s > best.score {
			best = escale{k, s, Chemin(prec, a.Case, k)}
			trouve = true
		}
	}
	return best, trouve
}

func (e *Etat) prixPleinEn(a *Avion, k Case) (int, int) {
	sauve := a.Case
	a.Case = k
	p, j := e.PrixPlein(a)
	a.Case = sauve
	return p, j
}

// ---------------------------------------------------------------------------
// Contrôleur

func (ia *IA) Decoller(e *Etat, a *Avion) bool {
	ia.init()
	budget := min(a.Jet, a.Carburant) - 2
	if ia.Temeraire {
		budget = a.Carburant - 2
	}
	es, ok := ia.meilleureEscale(e, a, budget)
	if !ok || es.score < 0.8 {
		delete(ia.plans, a.ID)
		return false
	}
	ia.plans[a.ID] = es.chemin
	return true
}

func (ia *IA) Voler(e *Etat, a *Avion) Ordre {
	ia.init()
	p := ia.plans[a.ID]
	if len(p) == 0 {
		if e.PeutSePoser(a) {
			return Ordre{Atterrir: true}
		}
		// plus de plan (avion de l'humain repris par l'IA, sursis) : l'escale la plus proche
		budget := a.AutonomieRestante()
		if budget <= 0 {
			budget = a.Carburant
		}
		es, ok := ia.meilleureEscale(e, a, min(budget, a.Carburant))
		if !ok {
			es, ok = ia.meilleureEscale(e, a, a.Carburant)
		}
		if !ok {
			return Ordre{Vers: e.CoupsVol(a)[0]}
		}
		p = es.chemin
	}
	ia.plans[a.ID] = p[1:]
	return Ordre{Vers: p[0]}
}

// CaseDeCrash : le terrain le plus clément (l'hydravion préfère un fleuve).
func (ia *IA) CaseDeCrash(e *Etat, a *Avion, choix []Case) Case {
	best, bs := choix[0], 99.0
	for _, k := range choix {
		s := float64(e.P.Terrain(k).Valeur())
		if a.Type == Hydravion && e.P.Fleuve(k) {
			s -= 2.5
		}
		if s < bs {
			best, bs = k, s
		}
	}
	return best
}

func (ia *IA) ChoisirAvion(e *Etat, j int, restants []*Avion) *Avion {
	return restants[0]
}

func (ia *IA) Escale(e *Etat, a *Avion) {
	p := e.PisteDe(a.Joueur, a.Case)
	if p != nil && len(p.Stock) > 0 {
		e.Reprendre(a)
	}
}

// Equiper : des moyen-courriers pour la navette (chaque escale rapporte un
// sac), puis des long-courriers pour l'Atlantique, et des pistes relais qui
// ouvrent de nouvelles villes au réseau des moyen-courriers.
func (ia *IA) Equiper(e *Etat, j int) {
	jr := &e.Joueurs[j]
	n := len(e.AvionsDe(j))
	reserve := 3000 + 1000*n
	for n < 5 {
		t := MoyenCourrier
		nlc := 0
		for _, a := range e.AvionsDe(j) {
			if a.Type != MoyenCourrier {
				nlc++
			}
		}
		if n >= 2 && nlc < n/2+1 {
			t = LongCourrier
			if ia.Audace > 0.5 && nlc >= 1 && e.PionsLibres(j, Hydravion) > 0 {
				t = Hydravion
			}
		}
		if e.PionsLibres(j, t) <= 0 {
			t = MoyenCourrier
		}
		if jr.Capital < t.Car().Prix+reserve {
			break
		}
		if _, err := e.Acheter(j, t); err != nil {
			break
		}
		n++
	}
	if jr.Capital >= 22000 && n >= 3 && len(e.PistesDe(j)) < 2 {
		if k, ok := ia.sitePiste(e, j); ok {
			if prix, err := e.DevisPiste(j, k); err == nil {
				if prix <= min(jr.Capital-5000, 7000+int(5000*ia.Audace)) {
					e.Construire(j, k, prix)
				}
			}
		}
	}
}

// joignables : villes du réseau des moyen-courriers depuis Paris, avec une
// escale de plus éventuelle.
func (e *Etat) joignables(extra Case) map[int]bool {
	nds := e.escales()
	if extra.Valide() {
		nds = append(nds, extra)
	}
	vu := map[Case]bool{e.P.CaseParis(): true}
	file := []Case{e.P.CaseParis()}
	m := sautMax(MoyenCourrier)
	for len(file) > 0 {
		u := file[0]
		file = file[1:]
		for _, n := range nds {
			if !vu[n] && Distance(u, n) <= m {
				vu[n] = true
				file = append(file, n)
			}
		}
	}
	res := map[int]bool{}
	for k := range vu {
		if v := e.P.Ville(k); v >= 0 {
			res[v] = true
		}
	}
	return res
}

// sitePiste : la terre libre qui ouvre le plus de villes aux moyen-courriers,
// ou qui borde une ville chère ; le moins cher possible.
func (ia *IA) sitePiste(e *Etat, j int) (Case, bool) {
	riche := e.Joueurs[j].Capital >= 30000
	avant := e.joignables(Nulle)
	best, bs := Nulle, 0.0
	for _, k := range Cases() {
		if e.PeutConstruire(j, k) != nil {
			continue
		}
		s, relais := 0.0, 0.0
		if riche {
			for _, n := range k.Voisins() {
				if v := e.P.Ville(n); v >= 0 && v != e.P.Paris {
					s += float64(e.P.Villes[v].Valeur) * 0.5
				}
			}
		}
		proche := false
		for _, n := range e.escales() {
			if Distance(k, n) <= sautMax(MoyenCourrier) && avant[e.P.Ville(n)] {
				proche = true
				break
			}
		}
		if proche {
			for v := range e.joignables(k) {
				if !avant[v] {
					g := float64(e.P.Villes[v].Valeur)
					if !e.Reliees[v] {
						g *= 1.5
					}
					relais += g
				}
			}
		}
		s += relais
		if s < 6 {
			continue
		}
		s /= float64(e.P.Terrain(k).Valeur())
		if s > bs {
			best, bs = k, s
		}
	}
	return best, best.Valide()
}
