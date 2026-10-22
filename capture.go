package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"aeropostale/jeu"
)

// Options de test : capture d'écran et mise en place directe d'une partie.
var (
	optCapture = flag.String("shot", "", "enregistre l'écran dans ce fichier PNG puis quitte")
	optImages  = flag.Int("frames", 30, "nombre d'images avant la capture")
	optEcran   = flag.String("ecran", "", "jeu : saute l'accueil")
	optTours   = flag.Int("tours", 0, "tours joués automatiquement avant de rendre la main")
	optGraine  = flag.Uint64("graine", 0, "graine du hasard (0 = aléatoire)")
	optCouleur = flag.Int("couleur", 0, "couleur du joueur (0 rouge, 1 jaune, 2 vert, 3 bleu)")
	optAdv     = flag.Int("adv", 2, "nombre d'adversaires (1 à 3)")
	optRegle   = flag.Int("regle", 0, "afficher cette page de la règle (1 à 6)")
	optRapide  = flag.Bool("rapide", false, "ordinateur accéléré")
	optPiste   = flag.Bool("piste", false, "captures : mode construction de piste")
	optSurvol  = flag.String("survol", "", "captures : souris sur cette case (c,r)")
	optTemer   = flag.Bool("temeraire", false, "captures : l'ordinateur qui joue pour vous vole au-delà de l'autonomie")
	optDevis   = flag.String("devis", "", "captures : demande le devis d'une piste sur cette case (c,r)")
	optPisteD  = flag.String("pistedepart", "", "captures : vous possédez d'office une piste sur cette case (c,r)")
	optArret   = flag.String("arret", "", "captures : l'ordinateur joue pour vous jusqu'à cette demande (equiper, devis, avion, decoller, voler, crash, escale)")
)

var arretCapture TypeReq

func (g *Game) preparer() {
	g.couleur = *optCouleur % 4
	g.nbAdv = min(max(*optAdv, 1), 3)
	if *optRegle > 0 && *optEcran == "" {
		g.accueilRegle, g.pageRegle = true, *optRegle-1
	}
	if *optEcran == "jeu" {
		arretCapture = map[string]TypeReq{"equiper": ReqEquiper, "devis": ReqDevis, "avion": ReqAvion,
			"decoller": ReqDecoller, "voler": ReqVoler, "crash": ReqCrash, "escale": ReqEscale}[*optArret]
		g.commencer(*optGraine, *optTours)
		if k, ok := caseOption(*optPisteD); ok {
			g.p.E.Pistes = append(g.p.E.Pistes, &jeu.Piste{Joueur: g.p.Humain, Case: k})
		}
		if *optTemer {
			g.p.E.Ctrl[g.p.Humain].(*Humain).ia.Temeraire = true
		}
		g.p.Rapide = *optRapide
		g.regles, g.pageRegle = *optRegle > 0, max(*optRegle-1, 0)
		g.modePiste = *optPiste
	}
}

func (g *Game) capturer(dst *ebiten.Image) {
	if *optCapture == "" {
		return
	}
	g.images++
	if g.images < *optImages {
		return
	}
	b := dst.Bounds()
	img := image.NewRGBA(b)
	dst.ReadPixels(img.Pix)
	f, err := os.Create(*optCapture)
	if err != nil {
		log.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()
	os.Exit(0)
}

func caseOption(s string) (jeu.Case, bool) {
	var k jeu.Case
	if s == "" {
		return k, false
	}
	fmt.Sscanf(s, "%d,%d", &k.C, &k.R)
	return k, k.Valide()
}

// clicCapture : option -devis, la case de la piste est choisie d'office.
func (g *Game) clicCapture() {
	if k, ok := caseOption(*optDevis); ok && g.p.Req.Type == ReqEquiper {
		*optDevis = ""
		g.p.repondre(cmdPiste{k})
	}
}

// survolCapture : option -survol, la souris est posée sur une case.
func (g *Game) survolCapture() {
	if *optSurvol == "" {
		return
	}
	var k jeu.Case
	fmt.Sscanf(*optSurvol, "%d,%d", &k.C, &k.R)
	if k.Valide() {
		g.survol = k
		x, y := centre(k)
		g.mx, g.my = int(x), int(y)
	}
}
