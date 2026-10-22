package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"aeropostale/jeu"
)

// Jetons dessinés une fois au démarrage : avions vus de dessus (nez vers la
// droite, on les tourne dans le sens du vol), pistes privées, enveloppe, dés.

const tailleAvion = 96

var (
	imgAvions   [jeu.NbTypes][4]*ebiten.Image
	imgPistes   [4]*ebiten.Image
	imgEnvel    *ebiten.Image
	imgEnvelVid *ebiten.Image
	imgDes      [7]*ebiten.Image
)

func construireJetons() {
	for t := range jeu.NbTypes {
		for c := range 4 {
			imgAvions[t][c] = dessinAvion(t, coulJoueurs[c])
		}
	}
	for c := range 4 {
		imgPistes[c] = dessinPiste(coulJoueurs[c])
	}
	imgEnvel = enveloppe(true)
	imgEnvelVid = enveloppe(false)
	for v := 1; v <= 6; v++ {
		imgDes[v] = de(v)
	}
}

// formes d'un avion dans un repère 0-100, nez vers +x
type forme struct {
	pts  [][2]float64
	rond bool // ellipse : pts[0] centre, pts[1] rayons
}

func ellipse(cx, cy, rx, ry float64) forme { return forme{[][2]float64{{cx, cy}, {rx, ry}}, true} }
func poly(p ...[2]float64) forme           { return forme{p, false} }

func formesAvion(t jeu.TypeAvion) (corps []forme, details []forme) {
	switch t {
	case jeu.MoyenCourrier: // monomoteur à aile haute (Latécoère 25)
		corps = []forme{
			ellipse(50, 50, 38, 6.5),
			poly([2]float64{44, 6}, [2]float64{60, 6}, [2]float64{58, 94}, [2]float64{44, 94}),
			poly([2]float64{12, 34}, [2]float64{22, 34}, [2]float64{24, 66}, [2]float64{12, 66}),
		}
		details = []forme{ellipse(86, 50, 2.5, 14), ellipse(60, 50, 4, 3.5)}
	case jeu.LongCourrier: // trimoteur (Latécoère 28)
		corps = []forme{
			ellipse(50, 50, 42, 7.5),
			poly([2]float64{42, 2}, [2]float64{60, 4}, [2]float64{60, 96}, [2]float64{42, 98}),
			poly([2]float64{8, 32}, [2]float64{19, 32}, [2]float64{21, 68}, [2]float64{8, 68}),
			ellipse(62, 24, 9, 4.5), ellipse(62, 76, 9, 4.5),
		}
		details = []forme{ellipse(92, 50, 2.5, 13), ellipse(71, 24, 2, 10), ellipse(71, 76, 2, 10), ellipse(62, 50, 4, 3.5)}
	case jeu.Hydravion: // hydravion à flotteurs, deux moteurs en tandem
		corps = []forme{
			ellipse(50, 30, 36, 5), ellipse(50, 70, 36, 5), // flotteurs
			ellipse(48, 50, 34, 6),
			poly([2]float64{42, 0}, [2]float64{58, 2}, [2]float64{58, 98}, [2]float64{42, 100}),
			poly([2]float64{10, 36}, [2]float64{20, 36}, [2]float64{22, 64}, [2]float64{10, 64}),
		}
		details = []forme{ellipse(52, 50, 10, 5), ellipse(64, 50, 2, 12), ellipse(38, 50, 2, 12)}
	}
	return
}

func chemin(f forme, s, o float64) *vector.Path {
	if f.rond {
		var p vector.Path
		cx, cy := f.pts[0][0]*s+o, f.pts[0][1]*s+o
		rx, ry := f.pts[1][0]*s, f.pts[1][1]*s
		const n = 40
		for i := range n {
			a := 2 * math.Pi * float64(i) / n
			x, y := float32(cx+rx*math.Cos(a)), float32(cy+ry*math.Sin(a))
			if i == 0 {
				p.MoveTo(x, y)
			} else {
				p.LineTo(x, y)
			}
		}
		p.Close()
		return &p
	}
	var pts [][2]float64
	for _, q := range f.pts {
		pts = append(pts, [2]float64{q[0]*s + o, q[1]*s + o})
	}
	return polygone(pts)
}

func dessinAvion(t jeu.TypeAvion, c color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(tailleAvion, tailleAvion)
	s, o := 0.82*tailleAvion/100, 0.09*tailleAvion
	corps, details := formesAvion(t)
	ombre := ebiten.NewImage(tailleAvion, tailleAvion)
	for _, f := range corps {
		tracer(ombre, chemin(f, s, o+3), 7, color.RGBA{0, 0, 0, 0x60})
		remplir(ombre, chemin(f, s, o+3), color.RGBA{0, 0, 0, 0x60})
	}
	img.DrawImage(ombre, nil)
	for _, f := range corps {
		tracer(img, chemin(f, s, o), 7, coulBlanc)
	}
	sombre := melange(c, coulEncre, 0.45)
	for _, f := range corps {
		remplir(img, chemin(f, s, o), c)
		tracer(img, chemin(f, s, o), 1.6, sombre)
	}
	for _, f := range details {
		remplir(img, chemin(f, s, o), coulEncre)
	}
	return img
}

// dessinPiste : pion carré de la couleur du joueur, une piste blanche en biais.
func dessinPiste(c color.RGBA) *ebiten.Image {
	const t = 64
	img := ebiten.NewImage(t, t)
	rectArrondi(img, 5, 5, t-10, t-10, 7, color.RGBA{0, 0, 0, 0x50})
	rectArrondi(img, 3, 3, t-10, t-10, 7, c)
	cadreArrondi(img, 3, 3, t-10, t-10, 7, 3, coulBlanc)
	m := t/2.0 - 2
	a := -math.Pi / 4
	ca, sa := math.Cos(a), math.Sin(a)
	L, W := 21.0, 6.5
	pts := [][2]float64{}
	for _, q := range [][2]float64{{-L, -W}, {L, -W}, {L, W}, {-L, W}} {
		pts = append(pts, [2]float64{m + q[0]*ca - q[1]*sa, m + q[0]*sa + q[1]*ca})
	}
	remplir(img, polygone(pts), coulEncre)
	for i := -2; i <= 2; i++ {
		x0, x1 := float64(i)*8-2.5, float64(i)*8+2.5
		ligne(img, m+x0*ca, m+x0*sa, m+x1*ca, m+x1*sa, 2, coulBlanc, true)
	}
	return img
}

func enveloppe(pleine bool) *ebiten.Image {
	img := ebiten.NewImage(30, 22)
	if pleine {
		rect(img, 1, 1, 28, 20, coulBlanc)
		// liseré « par avion »
		for i := 0; i < 28; i += 6 {
			remplir(img, polygone([][2]float64{{float64(1 + i), 1}, {float64(4 + i), 1}, {float64(1 + i), 4}}), coulRouge)
			remplir(img, polygone([][2]float64{{float64(4 + i), 21}, {float64(7 + i), 21}, {float64(7 + i), 18}}), coulBleu)
		}
		ligne(img, 2, 2, 15, 13, 1.6, coulSepia, true)
		ligne(img, 28, 2, 15, 13, 1.6, coulSepia, true)
		cadre(img, 1, 1, 28, 20, 1.5, coulSepia)
	} else {
		cadre(img, 1.5, 1.5, 27, 19, 1.5, coulPapier3)
		ligne(img, 2, 2, 15, 13, 1.4, coulPapier3, true)
		ligne(img, 28, 2, 15, 13, 1.4, coulPapier3, true)
	}
	return img
}

func de(v int) *ebiten.Image {
	const t = 64
	img := ebiten.NewImage(t, t)
	rectArrondi(img, 4, 6, t-8, t-8, 11, color.RGBA{0, 0, 0, 0x60})
	rectArrondi(img, 3, 3, t-8, t-8, 11, coulPapier)
	cadreArrondi(img, 3, 3, t-8, t-8, 11, 2.5, coulEncre)
	m := (t - 2.0) / 2
	d := 15.0
	pips := map[int][][2]float64{
		1: {{0, 0}},
		2: {{-1, -1}, {1, 1}},
		3: {{-1, -1}, {0, 0}, {1, 1}},
		4: {{-1, -1}, {1, -1}, {-1, 1}, {1, 1}},
		5: {{-1, -1}, {1, -1}, {0, 0}, {-1, 1}, {1, 1}},
		6: {{-1, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {1, 1}},
	}[v]
	for _, p := range pips {
		c := coulEncre
		if v == 1 {
			c = coulRouge
		}
		disque(img, m+p[0]*d, m+p[1]*d, 5.5, c, true)
	}
	return img
}
