package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/gen assets/polices
var fichiers embed.FS

var (
	imgCouv    *ebiten.Image // la couverture (accueil)
	imgTitre   *ebiten.Image // le titre de la règle (masque blanc)
	imgDessins [3]*ebiten.Image

	policeTitre, policeG, policeN, policeL, policeMachine *Police
)

// Palette : la poste aérienne des années 1930 — papier crème, encre marine,
// rouge et bleu des enveloppes « Par avion », or des affiches Art déco.
var (
	coulPapier  = color.RGBA{0xf4, 0xea, 0xd3, 0xff}
	coulPapier2 = color.RGBA{0xe6, 0xd6, 0xb2, 0xff}
	coulPapier3 = color.RGBA{0xd2, 0xbd, 0x93, 0xff}
	coulMarine  = color.RGBA{0x1b, 0x27, 0x45, 0xff}
	coulMarine2 = color.RGBA{0x2c, 0x3f, 0x68, 0xff}
	coulMarine3 = color.RGBA{0x4a, 0x5f, 0x8a, 0xff}
	coulEncre   = color.RGBA{0x2a, 0x22, 0x1c, 0xff}
	coulSepia   = color.RGBA{0x7a, 0x5a, 0x38, 0xff}
	coulRouge   = color.RGBA{0xbf, 0x2f, 0x26, 0xff}
	coulBleu    = color.RGBA{0x23, 0x55, 0xa0, 0xff}
	coulOr      = color.RGBA{0xd4, 0xa5, 0x2c, 0xff}
	coulOr2     = color.RGBA{0xf1, 0xd2, 0x7a, 0xff}
	coulBlanc   = color.RGBA{0xff, 0xff, 0xff, 0xff}
	coulTelegr  = color.RGBA{0xf7, 0xe7, 0xa8, 0xff}
	coulVertOK  = color.RGBA{0x3d, 0xa3, 0x4f, 0xff}
	coulOrange  = color.RGBA{0xf0, 0x8a, 0x1c, 0xff}
	coulDanger  = color.RGBA{0xd2, 0x26, 0x1c, 0xff}
	coulFond    = color.RGBA{0x14, 0x1d, 0x33, 0xff}
)

// couleurs des pions de l'encart : rouge, jaune, vert, bleu
var coulJoueurs = [4]color.RGBA{
	{0xcc, 0x33, 0x24, 0xff},
	{0xf0, 0xbc, 0x1c, 0xff},
	{0x27, 0x8f, 0x43, 0xff},
	{0x2a, 0x78, 0xcc, 0xff},
}

var nomsCouleurs = [4]string{"Rouge", "Jaune", "Vert", "Bleu"}

func chargerImage(nom string) *ebiten.Image {
	b, err := fichiers.ReadFile(nom)
	if err != nil {
		log.Fatal(err)
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		log.Fatal(nom, err)
	}
	return ebiten.NewImageFromImage(img)
}

// Police : une police (variable ou non) et sa graisse (0 : celle par défaut).
type Police struct {
	src   *text.GoTextFaceSource
	poids float32
}

var sources = map[string]*text.GoTextFaceSource{}

func chargerPolice(nom string, poids float32) *Police {
	if s, ok := sources[nom]; ok {
		return &Police{s, poids}
	}
	b, err := fichiers.ReadFile("assets/polices/" + nom)
	if err != nil {
		log.Fatal(err)
	}
	s, err := text.NewGoTextFaceSource(bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	sources[nom] = s
	return &Police{s, poids}
}

func chargerRessources() {
	imgCouv = chargerImage("assets/gen/couverture.jpg")
	imgTitre = chargerImage("assets/gen/titre.png")
	for i, n := range []string{"mc", "lc", "h"} {
		imgDessins[i] = chargerImage("assets/gen/regle_" + n + ".png")
	}
	policeTitre = chargerPolice("Limelight-Regular.ttf", 0)
	policeG = chargerPolice("JosefinSans.ttf", 700)
	policeN = chargerPolice("JosefinSans.ttf", 500)
	policeL = chargerPolice("JosefinSans.ttf", 400)
	policeMachine = chargerPolice("SpecialElite-Regular.ttf", 0)
	construirePlateau()
	construireJetons()
}

// ---------------------------------------------------------------- texte

var tagGraisse = text.MustParseTag("wght")

type cleFace struct {
	p *Police
	t float64
}

var faces = map[cleFace]*text.GoTextFace{}

func face(taille float64, p *Police) *text.GoTextFace {
	k := cleFace{p, taille}
	if f, ok := faces[k]; ok {
		return f
	}
	f := &text.GoTextFace{Source: p.src, Size: taille}
	if p.poids > 0 {
		f.SetVariation(tagGraisse, p.poids)
	}
	faces[k] = f
	return f
}

func ecrire(dst *ebiten.Image, s string, x, y float64, f text.Face, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, f, op)
}

func ecrireCentre(dst *ebiten.Image, s string, cx, y float64, f text.Face, c color.Color) {
	ecrire(dst, s, cx-text.Advance(s, f)/2, y, f, c)
}

func ecrireDroite(dst *ebiten.Image, s string, x, y float64, f text.Face, c color.Color) {
	ecrire(dst, s, x-text.Advance(s, f), y, f, c)
}

// ecrireHalo : texte cerné (noms sur le plateau).
func ecrireHalo(dst *ebiten.Image, s string, x, y float64, f text.Face, c, halo color.Color, ep float64) {
	for _, d := range [][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-.7, -.7}, {.7, .7}, {-.7, .7}, {.7, -.7}} {
		ecrire(dst, s, x+d[0]*ep, y+d[1]*ep, f, halo)
	}
	ecrire(dst, s, x, y, f, c)
}

func largeur(s string, f text.Face) float64 { return text.Advance(s, f) }

// mots : les mots d'un paragraphe ; une somme (« 12 000 F ») reste d'un bloc.
func mots(s string) []string {
	var l []string
	for _, m := range strings.Fields(s) {
		if n := len(l); n > 0 && (m == ";" || m == ":" || m == "!" || m == "?" || m == "»") {
			l[n-1] += " " + m
			continue
		}
		if n := len(l); n > 0 && finitParChiffre(l[n-1]) && (m == "F" || m == "F." || m == "F," || (len(m) >= 3 && estChiffre(m[0]) && estChiffre(m[1]) && estChiffre(m[2]))) {
			l[n-1] += " " + m
			continue
		}
		l = append(l, m)
	}
	return l
}

func estChiffre(c byte) bool { return c >= '0' && c <= '9' }

func finitParChiffre(s string) bool {
	return s != "" && (estChiffre(s[len(s)-1]) || strings.HasSuffix(s, "½"))
}

// couper : un texte en lignes de largeur maximale w.
func couper(s string, f text.Face, w float64) []string {
	var lignes []string
	for _, para := range strings.Split(s, "\n") {
		cur := ""
		for _, mot := range mots(para) {
			essai := mot
			if cur != "" {
				essai = cur + " " + mot
			}
			if text.Advance(essai, f) > w && cur != "" {
				lignes = append(lignes, cur)
				cur = mot
			} else {
				cur = essai
			}
		}
		lignes = append(lignes, cur)
	}
	return lignes
}

// ---------------------------------------------------------------- formes

func rect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func cadre(dst *ebiten.Image, x, y, w, h, ep float64, c color.Color) {
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), float32(ep), c, false)
}

func disque(dst *ebiten.Image, x, y, r float64, c color.Color, aa bool) {
	vector.FillCircle(dst, float32(x), float32(y), float32(r), c, aa)
}

func cercle(dst *ebiten.Image, x, y, r, ep float64, c color.Color, aa bool) {
	vector.StrokeCircle(dst, float32(x), float32(y), float32(r), float32(ep), c, aa)
}

func ligne(dst *ebiten.Image, x0, y0, x1, y1, ep float64, c color.Color, aa bool) {
	vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), float32(ep), c, aa)
}

func avecAlpha(c color.RGBA, a uint8) color.RGBA {
	f := float64(a) / 255
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), a}
}

func melange(a, b color.RGBA, t float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t + .5) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), m(a.A, b.A)}
}

// dessinerImage : img mise à l'échelle dans le rectangle (x, y, w, h).
func dessinerImage(dst, img *ebiten.Image, x, y, w, h float64, cs *ebiten.ColorScale) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(w/float64(img.Bounds().Dx()), h/float64(img.Bounds().Dy()))
	op.GeoM.Translate(x, y)
	if cs != nil {
		op.ColorScale = *cs
	}
	dst.DrawImage(img, op)
}

// dessinerCentre : img centrée en (cx, cy), à l'échelle s, tournée de a radians.
func dessinerCentre(dst, img *ebiten.Image, cx, cy, s, a float64, cs *ebiten.ColorScale) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Rotate(a)
	op.GeoM.Translate(cx, cy)
	if cs != nil {
		op.ColorScale = *cs
	}
	dst.DrawImage(img, op)
}

func teinte(c color.RGBA) *ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.ScaleWithColor(c)
	return &cs
}

func cheminArrondi(x, y, w, h, r float64) *vector.Path {
	r = min(r, w/2, h/2)
	X, Y, W, H, R := float32(x), float32(y), float32(w), float32(h), float32(r)
	var p vector.Path
	p.MoveTo(X+R, Y)
	p.LineTo(X+W-R, Y)
	p.ArcTo(X+W, Y, X+W, Y+R, R)
	p.LineTo(X+W, Y+H-R)
	p.ArcTo(X+W, Y+H, X+W-R, Y+H, R)
	p.LineTo(X+R, Y+H)
	p.ArcTo(X, Y+H, X, Y+H-R, R)
	p.LineTo(X, Y+R)
	p.ArcTo(X, Y, X+R, Y, R)
	p.Close()
	return &p
}

func remplir(dst *ebiten.Image, p *vector.Path, c color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.FillPath(dst, p, nil, op)
}

func tracer(dst *ebiten.Image, p *vector.Path, ep float64, c color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.StrokePath(dst, p, &vector.StrokeOptions{Width: float32(ep), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound}, op)
}

func rectArrondi(dst *ebiten.Image, x, y, w, h, r float64, c color.Color) {
	remplir(dst, cheminArrondi(x, y, w, h, r), c)
}

func cadreArrondi(dst *ebiten.Image, x, y, w, h, r, ep float64, c color.Color) {
	tracer(dst, cheminArrondi(x, y, w, h, r), ep, c)
}

func polygone(pts [][2]float64) *vector.Path {
	var p vector.Path
	for i, q := range pts {
		if i == 0 {
			p.MoveTo(float32(q[0]), float32(q[1]))
		} else {
			p.LineTo(float32(q[0]), float32(q[1]))
		}
	}
	p.Close()
	return &p
}

// sommetsHex : hexagone pointe en haut, de rayon r ; le côté k (sommets k
// et k+1) donne sur la direction 6-k (nord-est, est, sud-est…).
func sommetsHex(cx, cy, r float64) [][2]float64 {
	var s [][2]float64
	for k := range 6 {
		a := float64(k)*math.Pi/3 - math.Pi/2
		s = append(s, [2]float64{cx + r*math.Cos(a), cy + r*math.Sin(a)})
	}
	return s
}

func cosSin(a float64) (float64, float64) { return math.Cos(a), math.Sin(a) }
