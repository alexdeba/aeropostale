package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"aeropostale/jeu"
)

// ---------------------------------------------------------------- boutons

type StyleBouton int

const (
	BoutonNormal StyleBouton = iota
	BoutonGros
	BoutonClair // bouton crème sur papier
	BoutonPastille
)

type Bouton struct {
	X, Y, W, H float64
	Texte      string
	Sous       string // petite ligne sous le texte (prix…)
	Actif      bool
	Action     func()
	Style      StyleBouton
	Icone      *ebiten.Image
	Couleur    color.RGBA // pastilles
	Choisi     bool
}

func (b *Bouton) Contient(x, y int) bool {
	return float64(x) >= b.X && float64(x) < b.X+b.W && float64(y) >= b.Y && float64(y) < b.Y+b.H
}

// dessinerBouton : style Art déco, marine et or ; crème sur le papier.
func dessinerBouton(dst *ebiten.Image, b *Bouton, survol bool) {
	if b.Style == BoutonPastille {
		c := b.Couleur
		if b.Choisi {
			rectArrondi(dst, b.X-4, b.Y-4, b.W+8, b.H+8, 9, coulOr)
		}
		rectArrondi(dst, b.X+2, b.Y+3, b.W, b.H, 7, color.RGBA{0, 0, 0, 0x60})
		rectArrondi(dst, b.X, b.Y, b.W, b.H, 7, c)
		rectArrondi(dst, b.X+5, b.Y+5, b.W-10, b.H-10, 5, coulMer)
		if b.Icone != nil {
			dessinerCentre(dst, b.Icone, b.X+b.W/2, b.Y+b.H/2, (b.W-8)/float64(b.Icone.Bounds().Dx()), 0, nil)
		}
		if b.Texte != "" {
			f := face(18, policeG)
			ecrireCentre(dst, b.Texte, b.X+b.W/2, b.Y+b.H/2-11, f, coulBlanc)
		}
		return
	}
	fond, bord, txt := coulMarine2, coulOr, coulPapier
	if b.Style == BoutonClair {
		fond, bord, txt = coulPapier, coulMarine2, coulMarine
		if survol {
			fond = coulOr2
		}
	} else if survol {
		fond = coulMarine3
	}
	if !b.Actif {
		fond, bord, txt = melange(fond, coulPapier3, 0.55), coulPapier3, melange(txt, coulPapier3, 0.5)
	}
	if b.Choisi {
		fond, txt = coulOr, coulMarine
	}
	r := 6.0
	rectArrondi(dst, b.X+2, b.Y+3, b.W, b.H, r, color.RGBA{0, 0, 0, 0x50})
	rectArrondi(dst, b.X, b.Y, b.W, b.H, r, fond)
	cadreArrondi(dst, b.X+3, b.Y+3, b.W-6, b.H-6, r-2, 1.4, bord)
	taille := 15.0
	if b.Style == BoutonGros {
		taille = 21
	}
	f := face(taille, policeG)
	tx := b.X + b.W/2
	if b.Icone != nil {
		iw := b.H * 1.25
		dessinerImage(dst, b.Icone, b.X+8, b.Y+4, iw, b.H-8, teinte(txt))
		tx = b.X + 14 + iw + (b.W-iw-14)/2
	}
	if b.Sous != "" {
		ecrireCentre(dst, b.Texte, tx, b.Y+b.H/2-taille-1, f, txt)
		ecrireCentre(dst, b.Sous, tx, b.Y+b.H/2+2, face(13, policeN), melange(txt, fond, 0.25))
		return
	}
	ecrireCentre(dst, b.Texte, tx, b.Y+b.H/2-taille*0.62, f, txt)
}

// ---------------------------------------------------------------- panneau

const (
	px0 = PanneauX
	pw  = LargPanneau
)

func (g *Game) dessinerPanneau(dst *ebiten.Image) {
	e := g.p.E
	// papier à liseré « par avion »
	liserePostal(dst, px0-6, 8, pw+12, HauteurEcran-16)
	rect(dst, px0, 14, pw, HauteurEcran-28, coulPapier)
	// titre
	tw := pw - 40
	th := tw * float64(imgTitre.Bounds().Dy()) / float64(imgTitre.Bounds().Dx())
	dessinerImage(dst, imgTitre, px0+20, 24, tw, th, teinte(coulMarine))
	y := 24 + th + 6
	ecrireCentre(dst, "Courrier France – Afrique – Amérique du Sud", px0+pw/2, y, face(13, policeN), coulSepia)
	y += 22
	filet(dst, px0+16, y, pw-32)
	y += 8
	// tour et vent
	f := face(18, policeG)
	tour := fmt.Sprintf("Tour %d", e.Tour)
	ecrire(dst, tour, px0+18, y, f, coulMarine)
	if e.Vent > 0 {
		vx := px0 + 140.0
		ecrire(dst, "Vent vers "+versDirection(e.Vent), vx+30, y, face(16, policeN), coulMarine)
		fleche(dst, vx+12, y+11, angleDir(e.Vent), 12, coulRouge)
	}
	y += 30
	y = g.dessinerJoueurs(dst, y)
	y = g.dessinerConsigne(dst, y+6)
	g.yBoutons = y + 8
	g.dessinerFeuille(dst)
}

func versDirection(d int) string {
	n := jeu.NomDirection(d)
	if n == "est" || n == "ouest" {
		return "l'" + n
	}
	return "le " + n
}

// liserePostal : bordure à chevrons rouges et bleus des enveloppes par avion.
func liserePostal(dst *ebiten.Image, x, y, w, h float64) {
	rect(dst, x, y, w, h, coulPapier)
	const pas = 22.0
	ep := 6.0
	draw := func(x0, y0, x1, y1 float64, horizontal bool) {
		n := 0
		if horizontal {
			for t := x0; t < x1; t += pas {
				c := coulRouge
				if n%2 == 1 {
					c = coulBleu
				}
				remplir(dst, polygone([][2]float64{{t, y0}, {t + pas/2, y0}, {t + pas/2 + ep, y0 + ep}, {t + ep, y0 + ep}}), c)
				n++
			}
		} else {
			for t := y0; t < y1; t += pas {
				c := coulRouge
				if n%2 == 1 {
					c = coulBleu
				}
				remplir(dst, polygone([][2]float64{{x0, t}, {x0, t + pas/2}, {x0 + ep, t + pas/2 + ep}, {x0 + ep, t + ep}}), c)
				n++
			}
		}
	}
	draw(x, y, x+w, y, true)
	draw(x, y+h-ep, x+w, y+h, true)
	draw(x, y, x, y+h-ep, false)
	draw(x+w-ep, y, x+w, y+h-ep, false)
}

func filet(dst *ebiten.Image, x, y, w float64) {
	ligne(dst, x, y, x+w, y, 1.2, coulSepia, false)
	disque(dst, x+w/2, y, 3, coulRouge, true)
	ligne(dst, x+w/2-30, y+3, x+w/2+30, y+3, 1, coulSepia, false)
}

// fleche : petite flèche (les polices n'ont pas de flèche).
func fleche(dst *ebiten.Image, cx, cy, a, l float64, c color.RGBA) {
	ca, sa := cosSin(a)
	x0, y0 := cx-ca*l, cy-sa*l
	x1, y1 := cx+ca*l, cy+sa*l
	ligne(dst, x0, y0, x1-ca*4, y1-sa*4, 3, c, true)
	px, py := -sa, ca
	remplir(dst, polygone([][2]float64{{x1 + ca*2, y1 + sa*2}, {x1 - ca*7 + px*6, y1 - sa*7 + py*6}, {x1 - ca*7 - px*6, y1 - sa*7 - py*6}}), c)
}

// dessinerJoueurs : une fiche par compagnie.
func (g *Game) dessinerJoueurs(dst *ebiten.Image, y float64) float64 {
	e := g.p.E
	for _, j := range e.Ordre {
		jr := &e.Joueurs[j]
		h := 50.0
		c := coulJoueurs[jr.Couleur]
		if j == e.Courant && !e.Fini {
			rectArrondi(dst, px0+8, y-2, pw-16, h+2, 5, coulPapier2)
			remplir(dst, polygone([][2]float64{{px0 + 9, y + 12}, {px0 + 17, y + 19}, {px0 + 9, y + 26}}), coulRouge)
		}
		rectArrondi(dst, px0+22, y+4, 28, 28, 4, c)
		dessinerCentre(dst, imgAvions[jeu.MoyenCourrier][jr.Couleur], px0+36, y+18, 0.27, -0.6, nil)
		nom := jr.Nom
		if j == g.p.Humain {
			nom = "Vous"
		} else {
			nom = "Cie " + nom
		}
		ecrire(dst, nom, px0+58, y+2, face(16, policeG), coulEncre)
		ecrireDroite(dst, jeu.Francs(jr.Capital), px0+pw-14, y+2, face(16, policeG), coulMarine)
		// sacs rapportés : 13 enveloppes
		for i := range jeu.SacsVictoire {
			img := imgEnvelVid
			if i < jr.Rapportes {
				img = imgEnvel
			}
			dessinerImage(dst, img, px0+58+float64(i)*16, y+24, 15, 11, nil)
		}
		n := len(e.AvionsDe(j))
		info := fmt.Sprintf("%d sac(s) · %d avion(s) · %d piste(s)", jr.Rapportes, n, len(e.PistesDe(j)))
		if jr.Veterans > 0 {
			info += fmt.Sprintf(" · %d vétéran(s)", jr.Veterans)
		}
		ecrire(dst, info, px0+58, y+36, face(12, policeN), coulSepia)
		y += h + 4
	}
	return y
}

// consigne : le télégramme de la compagnie pour l'étape en cours.
func (g *Game) consigne() (titre, texte string) {
	p := g.p
	e := p.E
	r := p.Req
	if e.Fini || p.Termine {
		return "FIN DE PARTIE", "La partie est terminée"
	}
	switch r.Type {
	case ReqEquiper:
		if g.modePiste {
			return "CONSTRUCTION", "Cliquez la case de votre future piste privée (en or : terre libre, hors ville). Le prix sera tiré au dé : valeur du terrain × dé × 1 000 F"
		}
		return "ÉQUIPEMENT", fmt.Sprintf("Capital %s. Achetez un avion ou une piste privée, ou passez aux vols", jeu.Francs(e.Joueurs[p.Humain].Capital))
	case ReqDevis:
		t := e.P.Terrain(r.Case)
		return "DEVIS", fmt.Sprintf("Piste en %s (valeur %d) : le dé fixe le prix à %s. Construire ou renoncer ?", t, t.Valeur(), jeu.Francs(r.Prix))
	case ReqAvion:
		return "DÉPLACEMENTS", "Choisissez un avion à faire voler (cerclé d'or) : cliquez-le sur le plateau ou dans la liste. Le jet d'autonomie sera lancé"
	case ReqDecoller:
		a := r.Avion
		return "JET D'AUTONOMIE", fmt.Sprintf("%s : %s pt(s) en toute sécurité, %s pt(s) de carburant. Le décollage coûte 1 point. Cliquez une destination (vert : sûr, orange : risque de panne) ou restez au sol", a.Nom(), demi(a.Jet), demi(a.Carburant))
	case ReqVoler:
		a := r.Avion
		s := fmt.Sprintf("%s en vol : autonomie %s pt(s), carburant %s pt(s). Cliquez la case suivante ou une destination", a.Nom(), demi(max(a.AutonomieRestante(), 0)), demi(a.Carburant))
		if a.Conso >= a.Jet {
			s = fmt.Sprintf("%s : autonomie épuisée ! Chaque nouvelle case coûte un jet de panne (5-6 : le vol continue). Carburant %s pt(s). Posez-vous au plus vite", a.Nom(), demi(a.Carburant))
		}
		if e.PeutSePoser(a) {
			s += " — ou posez-vous ici"
		}
		return "EN VOL", s
	case ReqCrash:
		return "PANNE 3", "L'avion tombe ! Choisissez l'une des six cases voisines pour tenter de vous poser. Le pilote survit si son dé atteint la valeur du terrain (plaine 2, désert 3, jungle 4, montagne 5, île et mer 6)"
	case ReqEscale:
		return "VOTRE PISTE", "Vous pouvez entreposer votre courrier ici, ou reprendre celui qui y attend"
	}
	if e.Courant == p.Humain {
		return "EN COURS", "…"
	}
	return "CIE " + nomAffiche(e.Nom(e.Courant)), fmt.Sprintf("La compagnie de %s joue (phase de %s). Touche A : accélérer", e.Nom(e.Courant), e.Phase)
}

func (g *Game) dessinerConsigne(dst *ebiten.Image, y float64) float64 {
	titre, texte := g.consigne()
	f := face(16, policeMachine)
	w := pw - 24
	lignes := couper(texte+" STOP", f, w-28)
	erreur := g.p.Erreur
	var le []string
	if erreur != "" {
		le = couper(erreur, face(14, policeG), w-28)
	}
	h := 34 + float64(len(lignes))*20 + float64(len(le))*18 + 10
	x := px0 + 12
	rect(dst, x+2, y+3, w, h, color.RGBA{0, 0, 0, 0x40})
	rect(dst, x, y, w, h, coulTelegr)
	rect(dst, x, y, w, 26, coulMarine)
	ecrire(dst, "TÉLÉGRAMME", x+10, y+4, face(14, policeTitre), coulOr2)
	ecrireDroite(dst, titre, x+w-10, y+5, face(14, policeG), coulPapier)
	ty := y + 32
	for _, l := range lignes {
		ecrire(dst, l, x+14, ty, f, coulEncre)
		ty += 20
	}
	for _, l := range le {
		ecrire(dst, l, x+14, ty, face(14, policeG), coulDanger)
		ty += 18
	}
	return y + h
}

// boutonsJeu : selon la décision attendue.
func (g *Game) boutonsJeu() {
	p := g.p
	e := p.E
	r := p.Req
	x, w := px0+16, pw-32
	y := g.yBoutons
	if y == 0 {
		y = 470
	}
	j := p.Humain
	switch r.Type {
	case ReqEquiper:
		for t := range jeu.NbTypes {
			c := t.Car()
			b := g.bouton(x, y, w, 46, "Acheter un "+c.Nom, e.PeutAcheter(j, t) == nil, func() { p.repondre(cmdAcheter{t}) })
			b.Icone = imgDessins[t]
			b.Sous = fmt.Sprintf("%s · plein %d pts pour %s · %d dé(s)", jeu.Francs(c.Prix), c.Plein, jeu.Francs(c.PrixPlein), c.DesAutonomie)
			y += 52
		}
		b := g.bouton(x, y, w/2-4, 36, "Piste privée", true, func() { g.modePiste = !g.modePiste; p.Erreur = "" })
		b.Choisi = g.modePiste
		g.bouton(x+w/2+4, y, w/2-4, 36, "Passer aux vols", true, func() { p.repondre(cmdFin{}) })
		y += 42
	case ReqDevis:
		g.bouton(x, y, w/2-4, 40, "Construire", e.Joueurs[j].Capital >= r.Prix, func() { p.repondre(cmdOui{}) }).Sous = jeu.Francs(r.Prix)
		g.bouton(x+w/2+4, y, w/2-4, 40, "Renoncer", true, func() { p.repondre(cmdNon{}) })
		y += 46
	case ReqAvion:
		for i, a := range r.Avions {
			if i >= 4 {
				break
			}
			b := g.bouton(x, y, w, 34, fmt.Sprintf("%s — %s", a.Nom(), e.NomCase(a.Case)), true, func() { p.repondre(cmdAvion{a}) })
			b.Icone = imgDessins[a.Type]
			b.Style = BoutonClair
			y += 38
		}
		g.bouton(x, y, w, 36, "Terminer les vols du tour", true, func() { p.repondre(cmdFin{}) })
		y += 42
	case ReqDecoller:
		g.bouton(x, y, w, 36, "Rester au sol", true, func() { p.repondre(cmdNon{}) })
		y += 42
	case ReqVoler:
		if e.PeutSePoser(r.Avion) {
			g.bouton(x, y, w, 36, "Se poser ici", true, func() { p.repondre(cmdAtterrir{}) })
			y += 42
		}
	case ReqEscale:
		a := r.Avion
		pi := e.PisteDe(a.Joueur, a.Case)
		g.bouton(x, y, w/2-4, 36, fmt.Sprintf("Entreposer (%d)", len(a.Sacs)), len(a.Sacs) > 0, func() { p.repondre(cmdEntreposer{}) })
		n := 0
		if pi != nil {
			n = len(pi.Stock)
		}
		g.bouton(x+w/2+4, y, w/2-4, 36, fmt.Sprintf("Reprendre (%d)", n), n > 0, func() { p.repondre(cmdReprendre{}) })
		y += 42
		g.bouton(x, y, w, 34, "Continuer", true, func() { p.repondre(cmdFin{}) })
		y += 40
	}
	// barre du bas : la règle, la vitesse de l'ordinateur
	bw := (pw - 40) / 2
	b := g.bouton(px0+16, HauteurEcran-56, bw, 30, "La règle (R)", true, func() { g.regles = true })
	b.Style = BoutonClair
	t := "Accélérer (A)"
	if p.Rapide {
		t = "Ralentir (A)"
	}
	b = g.bouton(px0+24+bw, HauteurEcran-56, bw, 30, t, true, func() { p.Rapide = !p.Rapide })
	b.Style = BoutonClair
	g.yFeuille = y + 6
}

// avionMontre : l'avion dont on montre la feuille de route.
func (g *Game) avionMontre() *jeu.Avion {
	e := g.p.E
	if g.selAvion >= 0 && g.selAvion < len(e.Avions) {
		return e.Avions[g.selAvion]
	}
	if a := g.p.Req.Avion; a != nil {
		return a
	}
	if e.Actif >= 0 {
		return e.Avions[e.Actif]
	}
	if l := e.AvionsDe(g.p.Humain); len(l) > 0 {
		return l[0]
	}
	return nil
}

// dessinerFeuille : la feuille de route de l'avion (tableau 1 de la règle),
// puis le journal.
func (g *Game) dessinerFeuille(dst *ebiten.Image) {
	e := g.p.E
	y := g.yFeuille
	if y == 0 {
		y = 560
	}
	x, w := px0+12, pw-24
	bas := HauteurEcran - 106.0
	a := g.avionMontre()
	if a != nil && bas-y > 150 {
		h := min(bas-y-120, 300)
		h = max(h, 150)
		rect(dst, x+2, y+3, w, h, color.RGBA{0, 0, 0, 0x30})
		rect(dst, x, y, w, h, coulBlanc)
		cadre(dst, x, y, w, h, 2, coulRouge)
		jr := e.Joueurs[a.Joueur]
		ecrire(dst, "FEUILLE DE ROUTE", x+10, y+6, face(15, policeTitre), coulRouge)
		ecrireDroite(dst, map[bool]string{true: "Vous", false: "Cie " + jr.Nom}[a.Joueur == g.p.Humain], x+w-10, y+7, face(14, policeG), coulJoueurs[jr.Couleur])
		fm := face(14, policeMachine)
		ligneY := y + 28
		etat := "au sol, " + e.NomCase(a.Case)
		switch {
		case !a.Actif():
			etat = "détruit"
		case a.Etat == jeu.EnVol:
			etat = "en vol"
		case a.Reparation >= e.Tour:
			etat = "en réparations"
		}
		ecrire(dst, fmt.Sprintf("Type : %s   Numéro : %d", a.Type.Car().Court, a.Numero), x+10, ligneY, fm, coulEncre)
		pil := "pilote"
		if a.Veteran {
			pil = "vétéran (+1)"
		}
		ecrireDroite(dst, pil, x+w-10, ligneY, fm, coulEncre)
		ligneY += 18
		ecrire(dst, "État : "+etat, x+10, ligneY, fm, coulEncre)
		ligneY += 22
		// jauges
		jauge(dst, x+10, ligneY, w-20, "Carburant", a.Carburant, 2*a.Type.Car().Plein, coulMarine2, -1)
		ligneY += 26
		if a.Jet > 0 {
			jauge(dst, x+10, ligneY, w-20, "Autonomie", max(a.Jet-a.Conso, 0), a.Jet, coulVertOK, -1)
			ligneY += 26
		}
		// sacs à bord
		ecrire(dst, fmt.Sprintf("Courrier à bord : %d sac(s)", len(a.Sacs)), x+10, ligneY, fm, coulEncre)
		ligneY += 18
		sx := x + 10
		for _, s := range a.Sacs {
			v := e.P.Villes[s.Ville]
			et := fmt.Sprintf("%s %d", abrege(v.Nom), v.Valeur)
			lw := largeur(et, face(12, policeG)) + 10
			if sx+lw > x+w-8 {
				break
			}
			rectArrondi(dst, sx, ligneY, lw, 17, 3, coulPapier2)
			ecrire(dst, et, sx+5, ligneY+1, face(12, policeG), coulEncre)
			sx += lw + 4
		}
		if len(a.Sacs) > 0 {
			ligneY += 22
		}
		// les lignes de la feuille : n° tour | villes
		rect(dst, x+8, ligneY, w-16, 1.5, coulRouge)
		ecrire(dst, "Tour", x+10, ligneY+2, face(12, policeG), coulRouge)
		ecrire(dst, "Villes", x+52, ligneY+2, face(12, policeG), coulRouge)
		ligneY += 18
		rect(dst, x+46, ligneY-18, 1.5, y+h-ligneY+14, coulRouge)
		var lignes [][2]string
		for _, l := range a.Feuille {
			for i, s := range couper(l.Texte, face(13, policeMachine), w-66) {
				t := ""
				if i == 0 {
					t = fmt.Sprint(l.Tour)
				}
				lignes = append(lignes, [2]string{t, s})
			}
		}
		n := int((y + h - ligneY - 4) / 16)
		if len(lignes) > n {
			lignes = lignes[len(lignes)-n:]
		}
		for _, l := range lignes {
			ecrireDroite(dst, l[0], x+40, ligneY, face(13, policeMachine), coulEncre)
			ecrire(dst, l[1], x+52, ligneY, face(13, policeMachine), coulEncre)
			ligneY += 16
		}
		y += h + 10
	}
	// journal
	fj := face(12.5, policeN)
	var l []string
	for i := len(e.Journal) - 1; i >= 0 && len(l) < 14; i-- {
		c := couper(e.Journal[i], fj, w-8)
		for k := len(c) - 1; k >= 0; k-- {
			l = append(l, c[k])
		}
	}
	n := int((bas + 40 - y) / 16)
	if y+16 < bas-12 {
		filet(dst, x+20, y+2, w-150)
	}
	for i := 0; i < n && i < len(l); i++ {
		c := melange(coulEncre, coulPapier, float64(i)/float64(n+2))
		ecrire(dst, l[i], x+4, bas+40-float64(i+1)*16, fj, c)
	}
}

func abrege(n string) string {
	if strings.HasPrefix(n, "Îles ") {
		n = strings.TrimPrefix(n, "Îles ")
		n = strings.TrimPrefix(n, "du ")
	}
	if len([]rune(n)) > 10 {
		return string([]rune(n)[:9]) + "."
	}
	return n
}

// jauge : barre en points (demi-points).
func jauge(dst *ebiten.Image, x, y, w float64, nom string, v, maxi int, c color.RGBA, marque int) {
	fm := face(13, policeMachine)
	ecrire(dst, nom, x, y, fm, coulEncre)
	ecrireDroite(dst, fmt.Sprintf("%s / %s", demi(v), demi(maxi)), x+w, y, fm, coulEncre)
	bx, bw := x+90, w-160
	n := maxi / 2
	if n <= 0 {
		return
	}
	cw := bw / float64(n)
	for i := range n {
		cx := bx + float64(i)*cw
		rect(dst, cx+1, y+4, cw-2, 10, coulPapier2)
		reste := v - 2*i
		switch {
		case reste >= 2:
			rect(dst, cx+1, y+4, cw-2, 10, c)
		case reste == 1:
			rect(dst, cx+1, y+4, (cw-2)/2, 10, c)
		}
	}
	if maxi%2 == 1 {
		rect(dst, bx+float64(n)*cw+1, y+4, cw/2-2, 10, map[bool]color.RGBA{true: c, false: coulPapier2}[v == maxi])
	}
}
