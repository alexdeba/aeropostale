package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"aeropostale/jeu"
)

// ---------------------------------------------------------------- zone de vol

// zoneVol : cases accessibles par l'avion qui attend un ordre, avec le coût
// (demi-points, décollage compris) ; mise en cache tant que rien ne bouge.
type zoneVol struct {
	cle  [8]int
	cout [jeu.NbRang][jeu.NbCol]int
	prec [jeu.NbRang][jeu.NbCol]jeu.Case
	sol  bool // l'avion est au sol : le décollage coûte un point
	ok   bool
}

func (g *Game) zoneDe(e *jeu.Etat, r Requete) *zoneVol {
	a := r.Avion
	cle := [8]int{int(r.Type), a.ID, a.Case.C, a.Case.R, a.Conso, a.Carburant, e.Vent, a.Jet}
	if g.zone.ok && g.zone.cle == cle {
		return &g.zone
	}
	z := zoneVol{cle: cle, ok: true, sol: a.Etat == jeu.AuSol}
	budget := a.Carburant
	if z.sol {
		budget -= 2
	}
	z.cout, z.prec = e.Portee(a.Case, budget)
	if z.sol {
		for r := range jeu.NbRang {
			for c := range jeu.NbCol {
				if z.cout[r][c] > 0 {
					z.cout[r][c] += 2
				}
			}
		}
	}
	g.zone = z
	return &g.zone
}

// sur : la case est atteinte sans entrer en zone de panne.
func (z *zoneVol) sur(a *jeu.Avion, k jeu.Case) bool {
	c := z.cout[k.R][k.C]
	conso := a.Conso
	if z.sol {
		conso = 0
	}
	// le jet de panne survient au départ d'une case où l'autonomie est épuisée :
	// la case d'arrivée est sûre si l'on y entre avant épuisement
	return c > 0 && conso+c-dernierPas(z, a, k) < a.Jet
}

// dernierPas : coût de la dernière case du trajet.
func dernierPas(z *zoneVol, a *jeu.Avion, k jeu.Case) int {
	p := z.prec[k.R][k.C]
	if p == a.Case && z.sol {
		return z.cout[k.R][k.C] - 2
	}
	return z.cout[k.R][k.C] - z.cout[p.R][p.C]
}

// ---------------------------------------------------------------- glissements

type glissement struct {
	fx, fy, tx, ty float64
	t0             time.Time
}

const dureeGlisse = 0.17

func (g *Game) position(id int, tx, ty float64) (float64, float64) {
	gl, ok := g.glisse[id]
	now := time.Now()
	if !ok {
		g.glisse[id] = &glissement{tx, ty, tx, ty, now}
		return tx, ty
	}
	if gl.tx != tx || gl.ty != ty {
		cx, cy := gl.courant(now)
		*gl = glissement{cx, cy, tx, ty, now}
	}
	return gl.courant(now)
}

func (gl *glissement) courant(now time.Time) (float64, float64) {
	t := min(now.Sub(gl.t0).Seconds()/dureeGlisse, 1)
	t = t * t * (3 - 2*t)
	return gl.fx + (gl.tx-gl.fx)*t, gl.fy + (gl.ty-gl.fy)*t
}

// ---------------------------------------------------------------- plateau

func pulsation(g *Game, f float64) float64 {
	return 0.5 + 0.5*math.Sin(time.Since(g.t0).Seconds()*f)
}

func dessinerHex(dst *ebiten.Image, k jeu.Case, c color.RGBA, bord bool, echelle float64) {
	x, y := centre(k)
	img := imgHex
	if bord {
		img = imgHexBord
	}
	s := 2 * RayonHex / 98 * echelle
	dessinerCentre(dst, img, x, y, s, 0, teinte(c))
}

func (g *Game) dessinerPlateau(dst *ebiten.Image) {
	p := g.p
	e := p.E
	cadreTimbre(dst, PlateauX-14, PlateauY-14, LargPlat+28, HautPlat+28)
	dessinerImage(dst, imgPlateau, PlateauX, PlateauY, LargPlat, HautPlat, nil)
	// rose des vents et marqueur de tour
	if e.Vent > 0 {
		x, y := centre(caseRose)
		r := imgRoses[e.Vent]
		dessinerCentre(dst, r, x, y, 0.5, 0, nil)
	}
	if e.Tour > 0 {
		x, y := centre(casePiste(e.Tour))
		disque(dst, x+1.5, y+2, 15, color.RGBA{0, 0, 0, 0x60}, true)
		disque(dst, x, y, 15, coulBlanc, true)
		disque(dst, x, y, 12.5, coulEncre, true)
		ecrireCentre(dst, "T", x, y-11, face(19, policeG), coulBlanc)
	}
	g.dessinerSurbrillances(dst)
	// pistes privées
	for _, pi := range e.Pistes {
		x, y := centre(pi.Case)
		n := 0
		for _, q := range e.PistesSur(pi.Case) {
			if q == pi {
				break
			}
			n++
		}
		dessinerCentre(dst, imgPistes[e.Joueurs[pi.Joueur].Couleur], x+11-float64(n)*7, y+9-float64(n)*7, 0.36, 0, nil)
		if len(pi.Stock) > 0 {
			dessinerCentre(dst, imgEnvel, x+20, y+18, 0.55, 0, nil)
		}
	}
	g.dessinerAvions(dst)
	g.dessinerTrajet(dst)
	g.dessinerBandeau(dst)
	g.dessinerInfobulle(dst)
}

func (g *Game) dessinerSurbrillances(dst *ebiten.Image) {
	p := g.p
	e := p.E
	r := p.Req
	puls := pulsation(g, 5)
	switch r.Type {
	case ReqEquiper:
		if g.modePiste {
			for _, k := range jeu.Cases() {
				if e.PeutConstruire(p.Humain, k) != nil {
					dessinerHex(dst, k, color.RGBA{0x0c, 0x14, 0x28, 0x78}, false, 1.02)
				} else if k == g.survol {
					dessinerHex(dst, k, avecAlpha(coulOr2, 0x60), false, 1)
					dessinerHex(dst, k, coulOr, true, 1)
				}
			}
		}
	case ReqDevis:
		dessinerHex(dst, r.Case, avecAlpha(coulOr, uint8(120+100*puls)), true, 1)
	case ReqDecoller, ReqVoler:
		a := r.Avion
		z := g.zoneDe(e, r)
		for _, k := range jeu.Cases() {
			c := z.cout[k.R][k.C]
			switch {
			case k == a.Case:
			case c <= 0: // hors de portée : dans l'ombre
				dessinerHex(dst, k, color.RGBA{0x0c, 0x14, 0x28, 0x78}, false, 1.02)
			case z.sur(a, k):
				dessinerHex(dst, k, avecAlpha(color.RGBA{0xe8, 0xff, 0xd0, 0xff}, 0x40), false, 1)
			default:
				dessinerHex(dst, k, avecAlpha(coulOrange, 0x66), false, 1)
			}
		}
		for _, k := range jeu.Cases() {
			if z.cout[k.R][k.C] > 0 && e.PeutAtterrir(a, k) {
				col := coulBlanc
				if !z.sur(a, k) {
					col = coulOrange
				}
				dessinerHex(dst, k, avecAlpha(col, uint8(170+80*puls)), true, 0.92)
			}
		}
		dessinerHex(dst, a.Case, avecAlpha(coulOr, 0xff), true, 1)
	case ReqCrash:
		for _, k := range r.Choix {
			dessinerHex(dst, k, avecAlpha(coulDanger, uint8(150+100*puls)), true, 1)
			if k == g.survol {
				dessinerHex(dst, k, avecAlpha(coulDanger, 0x50), false, 1)
			}
		}
	}
}

// dessinerTrajet : trajet prévu jusqu'à la case survolée (points verts puis orange).
func (g *Game) dessinerTrajet(dst *ebiten.Image) {
	p := g.p
	r := p.Req
	if (r.Type != ReqDecoller && r.Type != ReqVoler) || !g.survol.Valide() {
		return
	}
	a := r.Avion
	z := g.zoneDe(p.E, r)
	k := g.survol
	if z.cout[k.R][k.C] <= 0 {
		return
	}
	ch := jeu.Chemin(z.prec, a.Case, k)
	px, py := centre(a.Case)
	for _, c := range ch {
		x, y := centre(c)
		col := coulVertOK
		if !z.sur(a, c) {
			col = coulOrange
		}
		for i := 1; i <= 4; i++ {
			t := float64(i) / 5
			dessinerCentre(dst, imgPoint, px+(x-px)*t, py+(y-py)*t, 0.5, 0, teinte(coulEncre))
			dessinerCentre(dst, imgPoint, px+(x-px)*t, py+(y-py)*t, 0.34, 0, teinte(col))
		}
		dessinerCentre(dst, imgPoint, x, y, 0.85, 0, teinte(coulEncre))
		dessinerCentre(dst, imgPoint, x, y, 0.62, 0, teinte(col))
		px, py = x, y
	}
}

// dessinerAvions : les avions, tournés dans le sens du vol, décalés quand
// plusieurs partagent une case.
func (g *Game) dessinerAvions(dst *ebiten.Image) {
	p := g.p
	e := p.E
	parCase := map[jeu.Case][]*jeu.Avion{}
	for _, a := range e.Avions {
		if a.Actif() {
			parCase[a.Case] = append(parCase[a.Case], a)
		}
	}
	actif := -1
	if p.Req.Avion != nil {
		actif = p.Req.Avion.ID
	} else if e.Actif >= 0 {
		actif = e.Actif
	}
	choix := map[int]bool{}
	if p.Req.Type == ReqAvion {
		for _, a := range p.Req.Avions {
			choix[a.ID] = true
		}
	}
	puls := pulsation(g, 5)
	var dernierDessin func()
	for k, l := range parCase {
		cx, cy := centre(k)
		for i, a := range l {
			ox, oy := 0.0, 0.0
			if len(l) > 1 {
				// en grappe autour de la case, la ville reste lisible au centre
				ang := float64(i)/float64(len(l))*2*math.Pi - math.Pi*0.75
				ox, oy = math.Cos(ang)*15, math.Sin(ang)*13
			}
			x, y := g.position(a.ID, cx+ox, cy+oy)
			cap := g.cap(a)
			a := a
			dessin := func() {
				if choix[a.ID] {
					cercle(dst, x, y, 20+3*puls, 3.5, coulOr, true)
				}
				if a.ID == actif || a.ID == g.selAvion {
					cercle(dst, x, y, 22, 2.5, coulBlanc, true)
				}
				var cs *ebiten.ColorScale
				if a.Reparation >= e.Tour {
					cs = &ebiten.ColorScale{}
					cs.Scale(0.55, 0.55, 0.55, 1)
				}
				dessinerCentre(dst, imgAvions[a.Type][e.Joueurs[a.Joueur].Couleur], x, y, 46.0/tailleAvion, cap, cs)
				if n := len(a.Sacs); n > 0 {
					bx, by := x+12, y+10
					dessinerCentre(dst, imgEnvel, bx, by, 0.6, 0, nil)
					ecrireHalo(dst, fmt.Sprint(n), bx+8, by-6, face(12, policeG), coulEncre, coulBlanc, 1.2)
				}
			}
			if a.ID == actif {
				dernierDessin = dessin
			} else {
				dessin()
			}
		}
	}
	if dernierDessin != nil {
		dernierDessin()
	}
}

// cap : sens du dernier déplacement (à l'est par défaut).
func (g *Game) cap(a *jeu.Avion) float64 {
	if n := len(a.Trajet); n >= 2 {
		x0, y0 := centre(a.Trajet[n-2])
		x1, y1 := centre(a.Trajet[n-1])
		if x0 != x1 || y0 != y1 {
			g.caps[a.ID] = math.Atan2(y1-y0, x1-x0)
		}
	}
	if c, ok := g.caps[a.ID]; ok {
		return c
	}
	return math.Pi // au départ de Paris, cap à l'ouest
}

// ---------------------------------------------------------------- bandeau

// dessinerBandeau : le dernier évènement (dés, pannes, livraisons) en
// télégramme au haut du plateau.
func (g *Game) dessinerBandeau(dst *ebiten.Image) {
	e := g.p.E
	ev := e.Dernier
	if ev == nil {
		return
	}
	if ev != g.dernier {
		g.dernier, g.depuis = ev, time.Now()
	}
	age := time.Since(g.depuis).Seconds()
	alpha := 1.0
	if age > 7 {
		alpha = max(0, 1-(age-7)/1.5)
	}
	if alpha <= 0 {
		return
	}
	f := face(17, policeMachine)
	w := 640.0
	lignes := couper(ev.Texte, f, w-40-float64(len(ev.Des))*46)
	h := max(24*float64(len(lignes))+22, 64)
	x := PlateauX + (LargPlat-w)/2
	y := PlateauY + HautPlat - 64 - h
	fond := coulTelegr
	if ev.Grave {
		fond = color.RGBA{0xf6, 0xc8, 0xb4, 0xff}
	}
	a8 := uint8(255 * alpha)
	rectArrondi(dst, x+3, y+4, w, h, 4, avecAlpha(color.RGBA{0, 0, 0, 0xff}, uint8(90*alpha)))
	rectArrondi(dst, x, y, w, h, 4, avecAlpha(fond, a8))
	rect(dst, x, y, 8, h, avecAlpha(coulRouge, a8))
	if ev.Joueur >= 0 {
		rect(dst, x, y, 8, h, avecAlpha(coulJoueurs[e.Joueurs[ev.Joueur].Couleur], a8))
	}
	ty := y + (h-24*float64(len(lignes)))/2
	for i, l := range lignes {
		ecrire(dst, l, x+22, ty+float64(i)*24, f, avecAlpha(coulEncre, a8))
	}
	var cs ebiten.ColorScale
	cs.ScaleAlpha(float32(alpha))
	for i, d := range ev.Des {
		dessinerCentre(dst, imgDes[d], x+w-34-float64(len(ev.Des)-1-i)*48, y+h/2, 0.66, 0, &cs)
	}
}

// ---------------------------------------------------------------- infobulle

func (g *Game) dessinerInfobulle(dst *ebiten.Image) {
	k := g.survol
	if !k.Valide() || g.regles {
		return
	}
	e := g.p.E
	var l []string
	if v := e.P.Ville(k); v >= 0 {
		vi := e.P.Villes[v]
		if v == e.P.Paris {
			l = append(l, "Paris — rapportez-y le courrier")
		} else {
			s := fmt.Sprintf("%s — sac : %s", vi.Nom, jeu.Francs(vi.Valeur*1000))
			if !e.Reliees[v] {
				s += " (×2 la 1re fois)"
			}
			l = append(l, s)
		}
	} else {
		t := e.P.Terrain(k)
		s := majuscule1(t.String())
		if t != jeu.Mer {
			s += fmt.Sprintf(" (valeur %d)", t.Valeur())
		}
		if e.P.Fleuve(k) {
			s += ", fleuve"
		}
		l = append(l, s)
	}
	for _, pi := range e.PistesSur(k) {
		s := "Piste de " + e.Nom(pi.Joueur)
		if len(pi.Stock) > 0 {
			s += fmt.Sprintf(" (%d sac(s) entreposé(s))", len(pi.Stock))
		}
		l = append(l, s)
	}
	for _, a := range e.Avions {
		if a.Actif() && a.Case == k {
			s := fmt.Sprintf("%s de %s", a.Nom(), e.Nom(a.Joueur))
			if n := len(a.Sacs); n > 0 {
				s += fmt.Sprintf(", %d sac(s)", n)
			}
			l = append(l, s)
		}
	}
	r := g.p.Req
	if r.Type == ReqEquiper && g.modePiste {
		if e.PeutConstruire(g.p.Humain, k) == nil {
			v := e.P.Terrain(k).Valeur()
			l = append(l, fmt.Sprintf("Piste privée : de %s à %s selon le dé", jeu.Francs(v*1000), jeu.Francs(v*6000)))
		} else if err := e.PeutConstruire(g.p.Humain, k); err != nil {
			l = append(l, "Impossible : "+err.Error())
		}
	}
	if (r.Type == ReqDecoller || r.Type == ReqVoler) && k != r.Avion.Case {
		z := g.zoneDe(e, r)
		if c := z.cout[k.R][k.C]; c > 0 {
			a := r.Avion
			s := fmt.Sprintf("Trajet : %s pt(s), il restera %s pt(s) de carburant", demi(c), demi(a.Carburant-c))
			if z.sol {
				s = fmt.Sprintf("Trajet : %s pt(s) décollage compris, il restera %s pt(s)", demi(c), demi(a.Carburant-c))
			}
			l = append(l, s)
			if !z.sur(a, k) {
				l = append(l, "Au-delà de l'autonomie : un jet de panne par case !")
			}
		}
	}
	f := face(15, policeN)
	w := 0.0
	for _, s := range l {
		w = max(w, largeur(s, f))
	}
	w += 20
	h := float64(len(l))*20 + 12
	x, y := float64(g.mx)+18, float64(g.my)+18
	if x+w > PlateauX+LargPlat {
		x = float64(g.mx) - w - 12
	}
	if y+h > HauteurEcran-8 {
		y = float64(g.my) - h - 12
	}
	rectArrondi(dst, x, y, w, h, 5, avecAlpha(coulMarine, 0xec))
	cadreArrondi(dst, x, y, w, h, 5, 1.5, coulOr)
	for i, s := range l {
		c := coulPapier
		if i > 0 && (s[0] == 'A' || s[0] == 'P' || s[0] == 'I') {
			c = coulOr2
		}
		ecrire(dst, s, x+10, y+6+float64(i)*20, f, c)
	}
}

// demi : 7 demi-points → « 3½ ».
func demi(n int) string {
	if n%2 == 0 {
		return fmt.Sprint(n / 2)
	}
	if n == 1 {
		return "½"
	}
	return fmt.Sprintf("%d½", n/2)
}

func majuscule1(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return nomAffiche(string(r[0])) + string(r[1:])
}

// ---------------------------------------------------------------- cadre

// cadreTimbre : le plateau est un timbre-poste, comme sur l'encart.
func cadreTimbre(dst *ebiten.Image, x, y, w, h float64) {
	rect(dst, x+4, y+5, w, h, color.RGBA{0, 0, 0, 0x70})
	rect(dst, x, y, w, h, coulPapier)
	const pas = 17.0
	for t := pas / 2; t < w; t += pas {
		disque(dst, x+t, y, 4.5, coulFond, true)
		disque(dst, x+t, y+h, 4.5, coulFond, true)
	}
	for t := pas / 2; t < h; t += pas {
		disque(dst, x, y+t, 4.5, coulFond, true)
		disque(dst, x+w, y+t, 4.5, coulFond, true)
	}
	cadre(dst, x+10, y+10, w-20, h-20, 1.5, color.RGBA{0x2c, 0x7a, 0x6a, 0xff})
}
