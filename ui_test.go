package main

import (
	"testing"
	"time"

	"aeropostale/jeu"
)

// Le contrôleur humain piloté comme le ferait l'interface : on attend chaque
// demande du moteur, on y répond par une commande.

func attendreReq(t *testing.T, p *Partie, types ...TypeReq) Requete {
	t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for time.Now().Before(limite) {
		p.mu.Lock()
		r := p.Req
		p.mu.Unlock()
		for _, ty := range types {
			if r.Type == ty {
				return r
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("demande %v jamais venue", types)
	return Requete{}
}

func repondre(p *Partie, v any) {
	p.mu.Lock()
	p.repondre(v)
	p.mu.Unlock()
}

func TestTourHumain(t *testing.T) {
	p := nouvellePartie(0, 1, 42, 0)
	p.muet = true
	p.lancer()
	attendreReq(t, p, ReqEquiper)
	// un moyen-courrier coûte 15 000 F : refusé avec 10 000 F
	repondre(p, cmdAcheter{jeu.MoyenCourrier})
	attendreReq(t, p, ReqEquiper)
	p.mu.Lock()
	if p.Erreur == "" || len(p.E.AvionsDe(p.Humain)) != 1 {
		t.Fatalf("achat accepté sans argent : %q", p.Erreur)
	}
	p.mu.Unlock()
	repondre(p, cmdFin{})
	r := attendreReq(t, p, ReqAvion)
	a := r.Avions[0]
	repondre(p, cmdAvion{a})
	// on vise la ville la plus proche à portée, ou on reste au sol
	vols := 0
	for {
		r := attendreReq(t, p, ReqDecoller, ReqAvion, ReqEquiper, ReqVoler)
		if r.Type != ReqDecoller {
			break
		}
		p.mu.Lock()
		e := p.E
		cout, _ := e.Portee(a.Case, a.Carburant-2)
		dest := jeu.Nulle
		for _, v := range e.P.Villes {
			k := v.Case
			if c := cout[k.R][k.C]; c > 0 && c+2 <= a.Jet && v.Valeur > 0 {
				dest = k
			}
		}
		p.mu.Unlock()
		if !dest.Valide() || vols > 0 {
			repondre(p, cmdNon{})
			break
		}
		vols++
		repondre(p, cmdVol{dest})
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if vols > 0 {
		if a.Etat != jeu.AuSol || e(p).P.Ville(a.Case) < 0 || len(a.Sacs) != 1 {
			t.Fatalf("après le vol : état %v, case %v, sacs %v", a.Etat, a.Case, a.Sacs)
		}
		if len(a.Feuille) == 0 {
			t.Fatal("feuille de route vide")
		}
	}
}

func e(p *Partie) *jeu.Etat { return p.E }

// TestCaptureAuto : les options de capture jouent pour l'humain jusqu'à la demande voulue.
func TestCaptureAuto(t *testing.T) {
	p := nouvellePartie(1, 2, 7, 3)
	p.muet = true
	p.arret = ReqDecoller
	p.jouerAuto()
	p.lancer()
	r := attendreReq(t, p, ReqDecoller)
	if r.Avion == nil || r.Avion.Joueur != p.Humain {
		t.Fatal("demande sans avion de l'humain")
	}
	p.mu.Lock()
	tour := p.E.Tour
	p.mu.Unlock()
	if tour <= 3 {
		t.Fatalf("tour %d : les tours automatiques n'ont pas été joués", tour)
	}
}

func TestCaseSous(t *testing.T) {
	for _, k := range jeu.Cases() {
		x, y := centre(k)
		if caseSous(x, y) != k || caseSous(x+RayonHex*0.7, y) != k {
			t.Fatalf("case %v mal retrouvée", k)
		}
	}
	if caseSous(2, 2).Valide() {
		t.Fatal("hors plateau")
	}
}
