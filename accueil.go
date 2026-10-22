package main

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"aeropostale/jeu"
)

// L'accueil : le plateau de l'encart en timbre-poste, le titre de la règle,
// le choix de la compagnie et du nombre d'adversaires.

func (g *Game) boutonsAccueil() {
	if g.accueilRegle {
		g.boutonsRegles()
		return
	}
	x, y := acPX+30, acPY
	for c := range 4 {
		b := g.bouton(x+float64(c)*92, y+258, 76, 76, "", true, func() { g.couleur = c })
		b.Style = BoutonPastille
		b.Couleur = coulJoueurs[c]
		b.Icone = imgAvions[jeu.LongCourrier][c]
		b.Choisi = g.couleur == c
	}
	for n := 1; n <= 3; n++ {
		b := g.bouton(x+float64(n-1)*124, y+388, 112, 46, fmt.Sprintf("%d", n), true, func() { g.nbAdv = n })
		b.Choisi = g.nbAdv == n
		b.Sous = map[int]string{1: "duel", 2: "à trois", 3: "à quatre"}[n]
	}
	b := g.bouton(x, y+474, 362, 64, "DÉCOLLER", true, func() { g.commencer(*optGraine, 0) })
	b.Style = BoutonGros
	b = g.bouton(x, y+554, 362, 40, "Lire la règle", true, func() { g.accueilRegle = true })
	b.Style = BoutonClair
}

// panneau des réglages, sous le titre de la couverture
const (
	acPX, acPY, acPW, acPH = 1140.0, 300.0, 422.0, 664.0
)

func (g *Game) dessinerAccueil(dst *ebiten.Image) {
	// la couverture à la hauteur de l'écran, centrée
	cw := float64(imgCouv.Bounds().Dx())
	dessinerImage(dst, imgCouv, (LargeurEcran-cw)/2, 0, cw, HauteurEcran, nil)
	// panneau des réglages : une enveloppe par avion
	x, y, w, h := acPX, acPY, acPW, acPH
	rect(dst, x+6, y+8, w, h, color.RGBA{0, 0, 0, 0x70})
	liserePostal(dst, x, y, w, h)
	rect(dst, x+8, y+8, w-16, h-16, coulPapier)
	ecrire(dst, "PAR AVION", x+30, y+26, face(30, policeTitre), coulBleu)
	ecrire(dst, "Ligne Paris – Dakar – Buenos Aires", x+30, y+70, face(16, policeMachine), coulEncre)
	texte := "Vous dirigez une compagnie d'aviation postale. Ramenez à Paris le courrier des villes d'Afrique et d'Amérique du Sud : le premier à 13 sacs l'emporte. Gare au vent, aux pannes et à la jungle !"
	for i, l := range couper(texte, face(16, policeN), w-60) {
		ecrire(dst, l, x+30, y+104+float64(i)*22, face(16, policeN), coulEncre)
	}
	ecrire(dst, "Votre compagnie", x+30, y+228, face(18, policeG), coulMarine)
	ecrire(dst, nomsCouleurs[g.couleur], x+30+largeur("Votre compagnie", face(18, policeG))+12, y+229, face(17, policeN), coulJoueurs[g.couleur])
	ecrire(dst, "Compagnies rivales", x+30, y+356, face(18, policeG), coulMarine)
	ecrireCentre(dst, "F. Nédélec · Jeux & Stratégie 1983 · dessin C. Lacroix", x+w/2, y+h-34, face(13, policeN), coulSepia)
}

// dessinerFin : vainqueur et classement.
func (g *Game) dessinerFin(dst *ebiten.Image) {
	e := g.p.E
	rect(dst, PlateauX, PlateauY, LargPlat, HautPlat, avecAlpha(coulFond, 0x90))
	w, h := 620.0, 230+44*float64(len(e.Joueurs))
	x, y := PlateauX+(LargPlat-w)/2, 640-h+30
	liserePostal(dst, x-8, y-8, w+16, h+16)
	rect(dst, x, y, w, h, coulPapier)
	gagnant := e.Gagnant
	titre := "Victoire !"
	if gagnant != g.p.Humain {
		titre = "La Cie " + e.Nom(gagnant) + " l'emporte"
	}
	ecrireCentre(dst, titre, x+w/2, y+30, face(36, policeTitre), coulRouge)
	ecrireCentre(dst, fmt.Sprintf("Tour %d", e.Tour), x+w/2, y+82, face(18, policeMachine), coulSepia)
	idx := make([]int, len(e.Joueurs))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool {
		ja, jb := e.Joueurs[idx[a]], e.Joueurs[idx[b]]
		if ja.Rapportes != jb.Rapportes {
			return ja.Rapportes > jb.Rapportes
		}
		return ja.Capital > jb.Capital
	})
	ty := y + 130
	for _, j := range idx {
		jr := e.Joueurs[j]
		nom := "Vous"
		if j != g.p.Humain {
			nom = "Cie " + jr.Nom
		}
		rectArrondi(dst, x+40, ty, 26, 26, 4, coulJoueurs[jr.Couleur])
		ecrire(dst, nom, x+80, ty+2, face(20, policeG), coulEncre)
		ecrire(dst, fmt.Sprintf("%d sac(s)", jr.Rapportes), x+300, ty+2, face(20, policeN), coulEncre)
		ecrireDroite(dst, jeu.Francs(jr.Capital), x+w-40, ty+2, face(20, policeN), coulMarine)
		ty += 44
	}
}
