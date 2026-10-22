package main

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// La règle consultable en jeu (touche R) : la règle de François Nédélec,
// résumée fidèlement, et les choix faits là où elle est muette.
// Format : « # » titre, « - » puce, « > » exemple, ligne vide = paragraphe.

var pagesRegle = []struct{ onglet, texte string }{
	{"Le jeu", `# Aéropostale
2 à 4 compagnies aériennes, au temps de Mermoz et de Saint-Exupéry, se disputent le courrier entre la France, l'Afrique et l'Amérique du Sud.
# But du jeu
Être le premier à avoir rapporté 13 sacs postaux à Paris, à la fin de son tour de jeu.
# Le début du jeu
Chaque joueur commence avec un avion moyen-courrier, le plein en carburant, et 10 000 F. Tous les avions partent de Paris.
# Le tour de jeu
- 1. Le marqueur T avance d'une case sur la bande numérotée.
- 2. Un dé fixe le vent du tour (1 à 6, la rose des vents du plateau) : le même pour tous.
Puis chaque joueur, à son tour, joue quatre phases :
- 3. Équipement : achat d'avions et d'aérodromes.
- 4. Déplacement de ses avions.
- 5. Prise du courrier (à chaque escale).
- 6. Ravitaillement.`},
	{"Équipement", `# Les avions
| Type | Prix | Plein | Points | Dés
| Moyen-courrier | 15 000 F | 1 000 F | 6 | 1 dé
| Long-courrier | 20 000 F | 3 000 F | 12 | 2 dés
| Hydravion | 25 000 F | 1 500 F | 8 | 2 dés
Un avion acheté attend à Paris, le plein fait. Chaque couleur a 6 moyen-courriers, 5 long-courriers et 4 hydravions.
# Les pistes privées
On choisit une case libre de terre (plaine, désert, jungle, montagne, île), sans ville ni aérodrome adverse. Le prix : un dé × valeur du terrain × 1 000 F, soit de 2 000 F à 36 000 F. Le dé lancé, on décide de construire ou non.
| Terrain | Valeur
| Plaine (vert clair) | 2
| Désert (jaune) | 3
| Jungle (vert foncé) | 4
| Montagne (marron) | 5
| Île | 6
Sur une île seulement, plusieurs joueurs peuvent avoir chacun leur piste (les pions s'empilent).
- Le propriétaire y fait le plein gratuitement ;
- les adversaires qui s'y posent paient le plein double, au propriétaire ;
- une piste voisine d'une ville permet d'en prendre le courrier ;
- on peut y entreposer son courrier.`},
	{"Déplacement", `# Le jet d'autonomie
Avant chaque décollage, on lance le jet d'autonomie : un dé pour le moyen-courrier, deux pour les autres (+1 pour un pilote vétéran). C'est le potentiel du vol en toute sécurité. On décide ensuite de décoller ou de rester au sol.
# La consommation
Le décollage coûte 1 point, l'atterrissage rien. Chaque case pénétrée coûte :
| Vent | Case | Montagne
| Arrière (sens du vent) | ½ | 1
| De côté (4 directions) | 1 | 2
| Debout (contre le vent) | 2 | 4
Les points sont pris à la fois sur l'autonomie et sur le carburant du plein.
# Les escales
On se pose sur une ville ou une piste privée. On peut faire autant d'escales qu'on veut : chaque nouveau décollage relance l'autonomie, tant qu'il reste du carburant. L'hydravion peut aussi se poser sur toutes les cases de fleuve.
# Le courrier
À chaque escale dans une ville (ou sur une piste voisine), l'avion prend un sac, noté sur sa feuille de route avec le numéro du tour.
# Ravitaillement
À la fin de ses vols, chaque avion posé sur une ville ou une piste refait le plein et le paie.`},
	{"Pannes", `# La panne
Si l'autonomie est épuisée en plein vol (ni ville ni piste), on lance un dé de panne avant d'avancer :
- 1 : crash mortel, l'avion et le pilote sont perdus ;
- 2 : crash sur la case ; jet de survie ;
- 3 : crash sur une des six cases voisines, au choix ; jet de survie ;
- 4 : atterrissage sans casse, mais un tour en réparations (c'est là qu'on rencontre le Petit Prince !) ; on repart ensuite avec le carburant restant ;
- 5-6 : pas de panne. Désormais l'avion n'avance plus que d'une case à la fois, avec un dé de panne à chaque case.
Sans plus de carburant en vol, c'est la panne irrémédiable : crash.
# Le jet de survie
Le pilote survit si son dé atteint la valeur du terrain où il s'est posé (plaine 2 … île 6). Pilote mort, avion perdu. Avion détruit mais pilote sauvé : il rentre à Paris et devient vétéran : +1 à chaque jet d'autonomie de son prochain avion.
# L'hydravion
Sur un fleuve, en cas de crash, l'hydravion n'est pas détruit sur un dé de 3, 4, 5 ou 6. Si le pilote survit aussi, ils repartent au tour suivant, ravitaillés.`},
	{"Gains", `# Les gains
Un sac rapporté à Paris rapporte 1 000 F × le chiffre de sa ville.
> Un sac de Cayenne (12) vaut 12 000 F.
# La première liaison
Le premier courrier rapporté d'une ville à Paris vaut double.
> Être le premier à rapporter un sac de Cayenne rapporte 24 000 F.
Si deux avions rapportent au même tour du courrier d'une ville reliée pour la première fois, la prime va au courrier le plus récent : celui pris au tour le plus élevé.
# La victoire
Le vainqueur est celui qui, à la fin de son tour de jeu, a rapporté 13 sacs de courrier (ou plus !).
# Exemple de feuille de route (tableau 1)
> Alger (1 sac), vent défavorable, Casablanca (1 sac), panne 4, Îles Canaries (1 sac), Casablanca (1 sac), Madrid (1 sac), puis Paris : 5 sacs, 25 000 F avec la prime des Canaries.`},
	{"Adaptation", `# Choix d'adaptation
La règle laisse quelques points dans l'ombre ; voici ceux qu'a tranchés ce programme.
- Le numéro du vent désigne la direction vers laquelle il souffle : voler dans ce sens, c'est avoir le vent arrière.
- Le jet de panne a lieu au moment de quitter une case où l'autonomie est épuisée : on peut donc encore se poser sur la ville atteinte.
- Un avion prend au plus un sac par ville et par tour. Paris ne fournit pas de courrier ; seule l'escale à Paris même livre les sacs.
- Panne 2 ou 3, ou panne sèche : l'avion est détruit (sauf l'hydravion sur un fleuve, sur 3 à 6) ; le jet de survie décide du pilote. En mer, il faut un 6.
- Panne 4 : l'avion passe le tour suivant au sol et repart celui d'après.
- La prime de première liaison double un seul sac ; elle est attribuée à la fin du tour.
- Une seule tentative de construction de piste par tour. Tout avion acheté a le plein ; le vétéran prend le prochain avion acheté.
- Le plein est obligatoire s'il est possible et payable ; sinon l'avion reste au sol avec ce qui lui reste.
- Pas d'alliances ni de commerce entre joueurs contre l'ordinateur.`},
}

const (
	rgX = PlateauX + 40
	rgY = PlateauY + 40
	rgW = LargPlat - 80
	rgH = HautPlat - 80
)

func (g *Game) touchesRegle() {
	n := len(pagesRegle)
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.pageRegle = (g.pageRegle + 1) % n
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.pageRegle = (g.pageRegle + n - 1) % n
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.regles, g.accueilRegle = false, false
	}
}

func (g *Game) boutonsRegles() {
	n := len(pagesRegle)
	w := (rgW - 60) / float64(n)
	for i, pg := range pagesRegle {
		b := g.bouton(rgX+30+float64(i)*w, rgY+100, w-8, 34, pg.onglet, true, func() { g.pageRegle = i })
		b.Choisi = i == g.pageRegle
	}
	b := g.bouton(rgX+rgW-140, rgY+rgH-54, 110, 36, "Fermer", true, func() { g.regles, g.accueilRegle = false, false })
	b.Style = BoutonClair
}

func (g *Game) dessinerRegles(dst *ebiten.Image) {
	rect(dst, 0, 0, LargeurEcran, HauteurEcran, avecAlpha(coulFond, 0xb0))
	liserePostal(dst, rgX-8, rgY-8, rgW+16, rgH+16)
	rect(dst, rgX, rgY, rgW, rgH, coulPapier)
	tw := 420.0
	th := tw * float64(imgTitre.Bounds().Dy()) / float64(imgTitre.Bounds().Dx())
	dessinerImage(dst, imgTitre, rgX+30, rgY+22, tw, th, teinte(coulMarine))
	ecrire(dst, "la règle de François Nédélec", rgX+470, rgY+38, face(20, policeMachine), coulSepia)
	ecrireDroite(dst, "flèches gauche et droite : pages", rgX+rgW-30, rgY+44, face(14, policeN), coulSepia)
	// colonnes de texte
	x0, y0 := rgX+40, rgY+154
	colW := (rgW - 120) / 2
	yMax := rgY + rgH - 70
	x, y := x0, y0
	ft := face(24, policeTitre)
	fn := face(17, policeN)
	fi := face(16, policeMachine)
	saut := func(h float64) {
		if y+h > yMax && x == x0 {
			x, y = x0+colW+40, y0
		}
	}
	for _, l := range strings.Split(pagesRegle[g.pageRegle].texte, "\n") {
		switch {
		case strings.HasPrefix(l, "# "):
			if y > y0 {
				y += 10
			}
			saut(60)
			ecrire(dst, strings.TrimPrefix(l, "# "), x, y, ft, coulRouge)
			y += 34
		case strings.HasPrefix(l, "| "):
			cells := strings.Split(strings.TrimPrefix(l, "| "), " | ")
			saut(26)
			premier := cells[0] == "Type" || cells[0] == "Terrain" || cells[0] == "Vent"
			if premier {
				rect(dst, x, y-2, colW, 25, coulMarine)
			} else {
				ligne(dst, x, y+23, x+colW, y+23, 1, coulPapier3, false)
			}
			pos := map[int][]float64{2: {0, .55}, 3: {0, .5, .75}, 5: {0, .36, .57, .78, .88}}[len(cells)]
			for i, c := range cells {
				f, col := fn, coulEncre
				if premier {
					f, col = face(15, policeG), coulPapier
				}
				ecrire(dst, c, x+8+colW*pos[i], y+1, f, col)
			}
			y += 26
		case strings.HasPrefix(l, "- "):
			lignes := couper(strings.TrimPrefix(l, "- "), fn, colW-22)
			saut(float64(len(lignes)) * 23)
			disque(dst, x+6, y+11, 3, coulRouge, true)
			for _, s := range lignes {
				ecrire(dst, s, x+18, y, fn, coulEncre)
				y += 23
			}
			y += 3
		case strings.HasPrefix(l, "> "):
			lignes := couper(strings.TrimPrefix(l, "> "), fi, colW-30)
			h := float64(len(lignes))*21 + 12
			saut(h)
			rect(dst, x, y, colW, h, coulTelegr)
			rect(dst, x, y, 5, h, coulBleu)
			for i, s := range lignes {
				ecrire(dst, s, x+16, y+6+float64(i)*21, fi, coulEncre)
			}
			y += h + 8
		default:
			lignes := couper(l, fn, colW)
			saut(float64(len(lignes)) * 23)
			for _, s := range lignes {
				ecrire(dst, s, x, y, fn, coulEncre)
				y += 23
			}
			y += 4
		}
	}
}
