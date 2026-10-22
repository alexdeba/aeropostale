package jeu

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func nouvelleIA4(n int, graine uint64) *Etat {
	noms := []string{"Rouge", "Jaune", "Vert", "Bleu"}[:n]
	var ctrls []Controleur
	for j := range n {
		ctrls = append(ctrls, NouvelleIA(0.3+0.2*float64(j)))
	}
	return Nouveau(noms, []int{0, 1, 2, 3}[:n], ctrls, graine)
}

func TestVoisinsSymetriques(t *testing.T) {
	for _, k := range Cases() {
		for d := 1; d <= 6; d++ {
			n := k.Voisin(d)
			if !n.Valide() {
				continue
			}
			if n.Voisin(Opposee(d)) != k {
				t.Fatalf("%v → %d → %v ne revient pas", k, d, n)
			}
			if Distance(k, n) != 1 {
				t.Fatalf("distance %v-%v = %d", k, n, Distance(k, n))
			}
		}
	}
}

func TestPlateau(t *testing.T) {
	p := NouveauPlateau()
	if len(p.Villes) != 27 {
		t.Fatalf("%d villes", len(p.Villes))
	}
	vu := map[Case]bool{}
	for _, v := range p.Villes {
		if !v.Case.Valide() || vu[v.Case] {
			t.Fatalf("ville %s mal placée %v", v.Nom, v.Case)
		}
		vu[v.Case] = true
		if p.Terrain(v.Case) == Mer {
			t.Errorf("%s en pleine mer", v.Nom)
		}
	}
	// les îles portent les villes insulaires
	for _, n := range []string{"Açores", "Îles Canaries", "Îles du Cap-Vert", "Ascension", "Sainte-Hélène"} {
		for _, v := range p.Villes {
			if v.Nom == n && p.Terrain(v.Case) != Ile {
				t.Errorf("%s : %s", n, p.Terrain(v.Case))
			}
		}
	}
	if p.Villes[p.Paris].Nom != "Paris" {
		t.Fatal("Paris")
	}
}

func TestCoutVent(t *testing.T) {
	e := nouvelleIA4(2, 1)
	k := Case{14, 20} // pleine mer
	e.Vent = Est
	if c := e.Cout(k, k.Voisin(Est)); c != 1 {
		t.Errorf("vent arrière %d", c)
	}
	if c := e.Cout(k, k.Voisin(Ouest)); c != 4 {
		t.Errorf("vent debout %d", c)
	}
	if c := e.Cout(k, k.Voisin(NordEst)); c != 2 {
		t.Errorf("vent de côté %d", c)
	}
	// montagne : le double
	for _, m := range Cases() {
		if e.P.Terrain(m) != Montagne {
			continue
		}
		for d := 1; d <= 6; d++ {
			n := m.Voisin(Opposee(d))
			if !n.Valide() {
				continue
			}
			want := map[bool]int{true: 2, false: 4}[d == e.Vent]
			if d == Opposee(e.Vent) {
				want = 8
			}
			if c := e.Cout(n, m); c != want {
				t.Fatalf("montagne dir %d : %d, attendu %d", d, c, want)
			}
		}
		break
	}
}

func TestDevisPiste(t *testing.T) {
	e := nouvelleIA4(2, 3)
	e.Phase, e.Courant = PhaseEquipement, 0
	e.Joueurs[0].Capital = 100000
	var plaine, ile, mer Case
	for _, k := range Cases() {
		if e.P.Ville(k) >= 0 {
			continue
		}
		switch e.P.Terrain(k) {
		case Plaine:
			plaine = k
		case Ile:
			ile = k
		case Mer:
			mer = k
		}
	}
	if e.PeutConstruire(0, mer) == nil {
		t.Error("piste en mer")
	}
	if e.PeutConstruire(0, e.P.CaseParis()) == nil {
		t.Error("piste sur une ville")
	}
	prix, err := e.DevisPiste(0, plaine)
	if err != nil || prix < 2000 || prix > 12000 || prix%2000 != 0 {
		t.Fatalf("devis plaine %d %v", prix, err)
	}
	if _, err := e.DevisPiste(0, plaine); err == nil {
		t.Error("deux devis le même tour")
	}
	e.Construire(0, plaine, prix)
	// île : les pistes s'empilent, pas ailleurs
	e.tentee = false
	e.Courant = 1
	e.Joueurs[1].Capital = 100000
	if e.PeutConstruire(1, plaine) == nil {
		t.Error("piste empilée en plaine")
	}
	if ile.Valide() {
		e.Pistes = append(e.Pistes, &Piste{Joueur: 0, Case: ile})
		if err := e.PeutConstruire(1, ile); err != nil {
			t.Errorf("piste empilée sur une île : %v", err)
		}
	}
}

func TestPrime(t *testing.T) {
	e := nouvelleIA4(2, 5)
	e.Tour = 7
	cay := -1
	for i, v := range e.P.Villes {
		if v.Nom == "Cayenne" {
			cay = i
		}
	}
	a, b := e.AvionsDe(0)[0], e.AvionsDe(1)[0]
	a.Sacs = []Sac{{cay, 3}}
	b.Sacs = []Sac{{cay, 5}}
	e.livrer(a)
	e.livrer(b)
	e.primes()
	// 12 000 F chacun, la prime (12 000 F) au courrier le plus récent
	if e.Joueurs[0].Capital != CapitalDepart+12000 || e.Joueurs[1].Capital != CapitalDepart+24000 {
		t.Fatalf("capitaux %d %d", e.Joueurs[0].Capital, e.Joueurs[1].Capital)
	}
	b.Sacs = []Sac{{cay, 6}}
	e.livrer(b)
	e.primes()
	if e.Joueurs[1].Capital != CapitalDepart+36000 {
		t.Fatalf("prime versée deux fois : %d", e.Joueurs[1].Capital)
	}
}

func TestCourrierPisteVoisine(t *testing.T) {
	e := nouvelleIA4(2, 9)
	e.Tour = 1
	a := e.AvionsDe(0)[0]
	var mad Ville
	for _, v := range e.P.Villes {
		if v.Nom == "Madrid" {
			mad = v
		}
	}
	var k Case
	for _, n := range mad.Case.Voisins() {
		if e.P.Terrain(n) != Mer && e.P.Ville(n) < 0 {
			k = n
		}
	}
	e.Pistes = append(e.Pistes, &Piste{Joueur: 1, Case: k})
	a.Case = k
	e.atterrir(a)
	if len(a.Sacs) != 1 || e.P.Villes[a.Sacs[0].Ville].Nom != "Madrid" {
		t.Fatalf("sacs %v", a.Sacs)
	}
	// un seul sac par ville et par tour
	a.Case = mad.Case
	e.atterrir(a)
	if len(a.Sacs) != 1 {
		t.Fatalf("deux sacs de Madrid le même tour")
	}
	// plein sur la piste adverse : le double, payé au propriétaire
	a.Case = k
	a.Carburant = 0
	c0, c1 := e.Joueurs[0].Capital, e.Joueurs[1].Capital
	e.ravitailler(0)
	if e.Joueurs[0].Capital != c0-2000 || e.Joueurs[1].Capital != c1+2000 {
		t.Fatalf("plein adverse %d %d", e.Joueurs[0].Capital-c0, e.Joueurs[1].Capital-c1)
	}
}

// TestParties : des parties entre ordinateurs se terminent par une victoire.
func TestParties(t *testing.T) {
	tours, victoires := 0, 0
	for g := range uint64(12) {
		e := nouvelleIA4(2+int(g%3), g+1)
		e.Partie()
		tours += e.Tour
		if e.Joueurs[e.Gagnant].Rapportes >= SacsVictoire {
			victoires++
		}
		for _, j := range e.Joueurs {
			if j.Capital < 0 {
				t.Fatalf("capital négatif %+v", j)
			}
		}
		t.Logf("graine %d : %d joueurs, %d tours, gagnant %s %+v", g+1, len(e.Joueurs), e.Tour, e.Nom(e.Gagnant), e.Joueurs)
	}
	if victoires < 10 {
		t.Errorf("seulement %d victoires sur 12", victoires)
	}
	t.Logf("%.1f tours en moyenne", float64(tours)/12)
}

// TestUne : UNE=graine,joueurs go test ./jeu -run TestUne -v
func TestUne(t *testing.T) {
	v := os.Getenv("UNE")
	if v == "" {
		t.Skip()
	}
	var g, n int
	fmt.Sscanf(v, "%d,%d", &g, &n)
	e := nouvelleIA4(n, uint64(g))
	e.Partie()
	fmt.Println(strings.Join(e.Journal, "\n"))
	for _, a := range e.Avions {
		fmt.Printf("%s %s : %+v\n", e.Nom(a.Joueur), a.Nom(), a.Feuille)
	}
}
