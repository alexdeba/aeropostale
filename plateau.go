package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"aeropostale/jeu"
)

// Géométrie du plateau redessiné : hexagones pointe en haut, pas de 40 px.
const (
	PlateauX = 26.0
	PlateauY = 26.0
	PasX     = 40.0
	PasY     = 34.641016 // PasX·√3/2
	RayonHex = 23.094011 // PasX/√3
	LargPlat = jeu.NbCol * PasX
	HautPlat = 2*RayonHex + (jeu.NbRang-1)*PasY
	kPlat    = 2 // le plateau est dessiné en double résolution
)

// centre d'une case à l'écran
func centre(k jeu.Case) (float64, float64) {
	x := PlateauX + PasX/2 + PasX*float64(k.C)
	if k.R%2 == 0 {
		x += PasX / 2
	}
	return x, PlateauY + RayonHex + PasY*float64(k.R)
}

// caseSous : la case sous le point (x, y), Nulle hors du plateau.
func caseSous(x, y float64) jeu.Case {
	r0 := int(math.Floor((y - PlateauY - RayonHex) / PasY))
	best, bd := jeu.Nulle, math.Inf(1)
	for r := r0 - 1; r <= r0+2; r++ {
		for c := -1; c <= jeu.NbCol; c++ {
			k := jeu.Case{C: c, R: r}
			if !k.Valide() {
				continue
			}
			cx, cy := centre(k)
			if d := math.Hypot(cx-x, cy-y); d < bd {
				best, bd = k, d
			}
		}
	}
	if bd > RayonHex {
		return jeu.Nulle
	}
	return best
}

var (
	imgPlateau *ebiten.Image    // le plateau redessiné (×2)
	imgRoses   [7]*ebiten.Image // la rose des vents, flèche du vent en couleur (×2)
	imgHex     *ebiten.Image    // hexagone blanc plein (×2), surbrillances
	imgHexBord *ebiten.Image    // son contour
	imgPoint   *ebiten.Image    // point blanc (trajets)
)

// couleurs des terrains (plateau de Claude Lacroix, éclaircies)
var (
	coulMer      = color.RGBA{0xa4, 0xd5, 0xea, 0xff}
	coulMer2     = color.RGBA{0x8c, 0xc5, 0xdf, 0xff}
	coulHautFond = color.RGBA{0xc6, 0xe8, 0xf3, 0xff}
	coulPlaine   = color.RGBA{0xa9, 0xc4, 0x3e, 0xff}
	coulPlaine2  = color.RGBA{0x6f, 0x93, 0x22, 0xff}
	coulDesert   = color.RGBA{0xf3, 0xc8, 0x45, 0xff}
	coulDesert2  = color.RGBA{0xd0, 0x8a, 0x1e, 0xff}
	coulJungle   = color.RGBA{0x2a, 0x86, 0x58, 0xff}
	coulJungle2  = color.RGBA{0x17, 0x5c, 0x3a, 0xff}
	coulJungle3  = color.RGBA{0x4c, 0xa8, 0x72, 0xff}
	coulMont     = color.RGBA{0x8b, 0x5a, 0x33, 0xff}
	coulMont2    = color.RGBA{0x55, 0x32, 0x18, 0xff}
	coulMont3    = color.RGBA{0xd9, 0xb3, 0x82, 0xff}
	coulCote     = color.RGBA{0x2c, 0x45, 0x34, 0xff}
	coulFleuve   = color.RGBA{0x1f, 0x8e, 0xc4, 0xff}
	coulGrille   = color.RGBA{0x1b, 0x27, 0x45, 0x50}
	coulRose     = color.RGBA{0x2f, 0x9c, 0xd6, 0xff}
)

func coulTerrain(t jeu.Terrain) color.RGBA {
	switch t {
	case jeu.Plaine, jeu.Ile:
		return coulPlaine
	case jeu.Desert:
		return coulDesert
	case jeu.Jungle:
		return coulJungle
	case jeu.Montagne:
		return coulMont
	}
	return coulMer
}

// hasard déterministe par case (textures)
func bruit(k jeu.Case, i int) float64 {
	h := uint32(k.C*73856093) ^ uint32(k.R*19349663) ^ uint32(i*83492791)
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return float64(h%10000) / 10000
}

var platTmp = jeu.NouveauPlateau()

func estTerre(k jeu.Case) bool {
	return k.Valide() && platTmp.Terrain(k) != jeu.Mer && platTmp.Terrain(k) != jeu.Ile
}

func construirePlateau() {
	p := platTmp
	const k = kPlat
	img := ebiten.NewImage(int(LargPlat*k), int(math.Ceil(HautPlat*k)))
	img.Fill(coulMer)
	ck := func(c jeu.Case) (float64, float64) {
		x, y := centre(c)
		return (x - PlateauX) * k, (y - PlateauY) * k
	}
	R := RayonHex * k
	// houle : petites vagues sur la mer
	for _, c := range jeu.Cases() {
		if estTerre(c) {
			continue
		}
		x, y := ck(c)
		for i := range 2 {
			vx := x + (bruit(c, i)-.5)*R*1.1
			vy := y + (bruit(c, i+5)-.5)*R*1.1
			vague(img, vx, vy, 7, coulMer2)
		}
	}
	// hauts-fonds autour des terres
	for _, c := range jeu.Cases() {
		if estTerre(c) {
			x, y := ck(c)
			remplir(img, polygone(sommetsHex(x, y, R*1.22)), coulHautFond)
		}
	}
	// terres
	for _, c := range jeu.Cases() {
		t := p.Terrain(c)
		x, y := ck(c)
		switch {
		case t == jeu.Ile:
			ile(img, c, x, y, R)
		case estTerre(c):
			remplir(img, polygone(sommetsHex(x, y, R+0.8)), coulTerrain(t))
			texture(img, c, t, x, y, R)
		}
	}
	// grille
	for _, c := range jeu.Cases() {
		x, y := ck(c)
		tracer(img, polygone(sommetsHex(x, y, R)), 1.6, coulGrille)
	}
	// côtes : les côtés d'une terre qui donnent sur la mer
	for _, c := range jeu.Cases() {
		if !estTerre(c) {
			continue
		}
		x, y := ck(c)
		s := sommetsHex(x, y, R)
		for e := range 6 {
			n := c.Voisin(6 - e)
			if n.Valide() && estTerre(n) {
				continue
			}
			a, b := s[e], s[(e+1)%6]
			ligne(img, a[0], a[1], b[0], b[1], 3.4, coulCote, true)
		}
	}
	fleuves(img, ck)
	piste(img, ck)
	rose(img, ck)
	villes(img, ck)
	// le titre, comme sur l'encart, en lettres blanches en haut de la mer
	tw := 15 * PasX * k
	th := tw * float64(imgTitre.Bounds().Dy()) / float64(imgTitre.Bounds().Dx())
	tx := 7.6 * PasX * k
	sh := teinte(color.RGBA{0x1b, 0x27, 0x45, 0x70})
	dessinerImage(img, imgTitre, tx+5, 14+5, tw, th, sh)
	dessinerImage(img, imgTitre, tx, 14, tw, th, teinte(color.RGBA{0xfb, 0xfd, 0xff, 0xff}))
	imgPlateau = img

	imgHex = ebiten.NewImage(100, 100)
	remplir(imgHex, polygone(sommetsHex(50, 50, 49)), coulBlanc)
	imgHexBord = ebiten.NewImage(100, 100)
	tracer(imgHexBord, polygone(sommetsHex(50, 50, 45.5)), 6, coulBlanc)
	imgPoint = ebiten.NewImage(16, 16)
	disque(imgPoint, 8, 8, 7.5, coulBlanc, true)
	for v := range 7 {
		imgRoses[v] = ebiten.NewImage(int(3.6*PasX*k), int(3.6*PasX*k))
		roseVents(imgRoses[v], v)
	}
}

func vague(img *ebiten.Image, x, y, l float64, c color.RGBA) {
	var p vector.Path
	p.MoveTo(float32(x-l), float32(y))
	p.QuadTo(float32(x-l/2), float32(y-l/2.4), float32(x), float32(y))
	p.QuadTo(float32(x+l/2), float32(y-l/2.4), float32(x+l), float32(y))
	tracer(img, &p, 1.6, c)
}

// ile : un îlot vert cerné dans une case de mer.
func ile(img *ebiten.Image, c jeu.Case, x, y, R float64) {
	n := 2 + int(bruit(c, 9)*3)
	for i := range n {
		ix := x + (bruit(c, i)-.5)*R*1.0
		iy := y + (bruit(c, i+3)-.5)*R*0.8
		r := R * (0.12 + 0.12*bruit(c, i+6))
		disque(img, ix, iy, r*1.9, coulHautFond, true)
	}
	for i := range n {
		ix := x + (bruit(c, i)-.5)*R*1.0
		iy := y + (bruit(c, i+3)-.5)*R*0.8
		r := R * (0.12 + 0.12*bruit(c, i+6))
		var p vector.Path
		p.MoveTo(float32(ix-r*1.3), float32(iy))
		p.QuadTo(float32(ix-r), float32(iy-r*1.1), float32(ix+r*0.2), float32(iy-r*0.8))
		p.QuadTo(float32(ix+r*1.6), float32(iy-r*0.4), float32(ix+r*1.2), float32(iy+r*0.5))
		p.QuadTo(float32(ix), float32(iy+r*1.1), float32(ix-r*1.3), float32(iy))
		p.Close()
		remplir(img, &p, coulPlaine)
		tracer(img, &p, 2, coulCote)
	}
}

// texture : herbe, dunes, frondaisons, sommets.
func texture(img *ebiten.Image, c jeu.Case, t jeu.Terrain, x, y, R float64) {
	pos := func(i int, f float64) (float64, float64) {
		return x + (bruit(c, i)-.5)*R*f, y + (bruit(c, i+11)-.5)*R*f
	}
	switch t {
	case jeu.Plaine:
		for i := range 7 {
			px, py := pos(i, 1.3)
			for d := -1.0; d <= 1; d++ {
				ligne(img, px+d*3, py, px+d*4.5, py-6, 1.5, coulPlaine2, true)
			}
		}
	case jeu.Desert:
		for i := range 4 {
			px, py := pos(i, 1.1)
			var p vector.Path
			p.MoveTo(float32(px-12), float32(py+2))
			p.QuadTo(float32(px-4), float32(py-7), float32(px+2), float32(py))
			p.QuadTo(float32(px+8), float32(py+6), float32(px+14), float32(py-1))
			tracer(img, &p, 2, coulDesert2)
		}
	case jeu.Jungle:
		for i := range 9 {
			px, py := pos(i, 1.35)
			r := 7 + 4*bruit(c, i+20)
			disque(img, px+1.5, py+2, r, coulJungle2, true)
			disque(img, px, py, r, coulJungle, true)
			disque(img, px-r*0.3, py-r*0.3, r*0.45, coulJungle3, true)
		}
	case jeu.Montagne:
		for i := range 3 {
			px, py := pos(i, 0.9)
			h := 16 + 8*bruit(c, i+30)
			l := h * 0.95
			remplir(img, polygone([][2]float64{{px - l, py + h/2}, {px, py - h/2}, {px + l, py + h/2}}), coulMont2)
			remplir(img, polygone([][2]float64{{px - l*0.55, py + h/2}, {px, py - h/2}, {px + l*0.15, py + h/2}}), coulMont)
			remplir(img, polygone([][2]float64{{px - l*0.3, py - h/2 + h*0.3}, {px, py - h/2}, {px + l*0.3, py - h/2 + h*0.3}}), coulMont3)
		}
	}
}

// fleuves : les cases de fleuve voisines sont reliées (arbre couvrant), les
// bouts qui touchent la mer s'y jettent.
func fleuves(img *ebiten.Image, ck func(jeu.Case) (float64, float64)) {
	p := platTmp
	vu := map[jeu.Case]bool{}
	type seg struct{ a, b [2]float64 }
	var segs []seg
	deg := map[jeu.Case]int{}
	for _, c := range jeu.Cases() {
		if !p.Fleuve(c) || vu[c] {
			continue
		}
		vu[c] = true
		file := []jeu.Case{c}
		for len(file) > 0 {
			x := file[0]
			file = file[1:]
			for _, n := range x.Voisins() {
				if p.Fleuve(n) && !vu[n] {
					vu[n] = true
					file = append(file, n)
					ax, ay := ck(x)
					bx, by := ck(n)
					segs = append(segs, seg{[2]float64{ax, ay}, [2]float64{bx, by}})
					deg[x]++
					deg[n]++
				}
			}
		}
	}
	for _, c := range jeu.Cases() {
		if !p.Fleuve(c) || deg[c] > 1 {
			continue
		}
		for _, n := range c.Voisins() {
			if !estTerre(n) {
				ax, ay := ck(c)
				bx, by := ck(n)
				segs = append(segs, seg{[2]float64{ax, ay}, [2]float64{(ax + bx) / 2, (ay + by) / 2}})
				break
			}
		}
	}
	for _, s := range segs {
		// léger méandre : point milieu décalé
		mx, my := (s.a[0]+s.b[0])/2, (s.a[1]+s.b[1])/2
		dx, dy := s.b[0]-s.a[0], s.b[1]-s.a[1]
		o := (bruit(jeu.Case{C: int(mx), R: int(my)}, 1) - .5) * 0.5
		mx, my = mx-dy*o, my+dx*o
		var pa vector.Path
		pa.MoveTo(float32(s.a[0]), float32(s.a[1]))
		pa.QuadTo(float32(mx), float32(my), float32(s.b[0]), float32(s.b[1]))
		tracer(img, &pa, 7, color.RGBA{0x16, 0x5f, 0x88, 0xff})
		tracer(img, &pa, 4.5, coulFleuve)
	}
}

// piste : la bande des tours au bas de l'encart.
const debutPiste = 2 // colonne de la case « 1 » (rangée 26)

func casePiste(t int) jeu.Case {
	return jeu.Case{C: debutPiste + (t-1)%20, R: jeu.NbRang - 1}
}

func piste(img *ebiten.Image, ck func(jeu.Case) (float64, float64)) {
	f := face(19*kPlat, policeG)
	for t := 1; t <= 20; t++ {
		x, y := ck(casePiste(t))
		remplir(img, polygone(sommetsHex(x, y, RayonHex*kPlat-1)), color.RGBA{0x1e, 0x7f, 0xc4, 0xff})
		tracer(img, polygone(sommetsHex(x, y, RayonHex*kPlat-5)), 2, color.RGBA{0x8f, 0xcd, 0xf0, 0xff})
		s := fmt.Sprint(t)
		ecrireCentre(img, s, x, y-23, f, coulBlanc)
	}
}

// la rose des vents de l'encart, case (10, 4)
var caseRose = jeu.Case{C: 10, R: 4}

func rose(img *ebiten.Image, ck func(jeu.Case) (float64, float64)) {
	x, y := ck(caseRose)
	roseDessin(img, x, y, 0, coulRose)
}

// roseVents : la rose seule, la flèche du vent v en rouge (0 : aucune).
func roseVents(img *ebiten.Image, v int) {
	w := float64(img.Bounds().Dx())
	roseDessin(img, w/2, w/2, v, coulRouge)
}

// angle de la direction d (1 nord-ouest… 6 nord-est), en radians écran
func angleDir(d int) float64 {
	return map[int]float64{1: -120, 2: 180, 3: 120, 4: 60, 5: 0, 6: -60}[d] * math.Pi / 180
}

func roseDessin(img *ebiten.Image, x, y float64, v int, c color.RGBA) {
	L := 1.55 * PasX * kPlat
	f := face(15*kPlat, policeG)
	for d := 1; d <= 6; d++ {
		if v != 0 && d != v {
			continue
		}
		a := angleDir(d)
		ca, sa := math.Cos(a), math.Sin(a)
		px, py := -sa, ca
		ep := 3.0
		if v != 0 {
			ep = 7
		}
		x0, y0 := x+ca*L*0.22, y+sa*L*0.22
		x1, y1 := x+ca*L*0.82, y+sa*L*0.82
		ligne(img, x0, y0, x1, y1, ep, c, true)
		// pointe
		t := [][2]float64{{x + ca*L, y + sa*L}, {x1 + px*12, y1 + py*12}, {x1 - px*12, y1 - py*12}}
		if v != 0 {
			t = [][2]float64{{x + ca*L*1.02, y + sa*L*1.02}, {x1 + px*20, y1 + py*20}, {x1 - px*20, y1 - py*20}}
			remplir(img, polygone(t), c)
		} else {
			tracer(img, polygone(t), 3, c)
		}
		if v == 0 {
			s := fmt.Sprint(d)
			nx, ny := x+ca*L*1.2, y+sa*L*1.2
			ecrireCentre(img, s, nx, ny-20, f, c)
		}
	}
	if v == 0 {
		// Éole : un nuage joufflu au centre
		for i := range 6 {
			a := float64(i) * math.Pi / 3
			disque(img, x+math.Cos(a)*16, y+math.Sin(a)*14, 15, color.RGBA{0xe4, 0xf4, 0xfb, 0xff}, true)
		}
		disque(img, x, y, 22, color.RGBA{0xe4, 0xf4, 0xfb, 0xff}, true)
		for i := range 6 {
			a := float64(i) * math.Pi / 3
			cercle(img, x+math.Cos(a)*16, y+math.Sin(a)*14, 15, 2.5, c, true)
		}
		disque(img, x, y, 21, color.RGBA{0xe4, 0xf4, 0xfb, 0xff}, true)
		disque(img, x-7, y-4, 2.6, c, true)
		disque(img, x+7, y-4, 2.6, c, true)
		cercle(img, x, y+7, 4.5, 2.4, c, true)
	}
}

// villes : disque blanc à la valeur du sac, nom en blanc cerné.
func villes(img *ebiten.Image, ck func(jeu.Case) (float64, float64)) {
	fv := face(15*kPlat, policeG)
	fn := face(11.5*kPlat, policeG)
	for i, v := range platTmp.Villes {
		x, y := ck(v.Case)
		if i == platTmp.Paris {
			// cocarde tricolore
			disque(img, x, y-4, 21, coulBlanc, true)
			disque(img, x, y-4, 18, color.RGBA{0x23, 0x45, 0x9a, 0xff}, true)
			disque(img, x, y-4, 12.5, coulBlanc, true)
			disque(img, x, y-4, 7, coulRouge, true)
			cercle(img, x, y-4, 21, 2, coulEncre, true)
		} else {
			disque(img, x+1.5, y-6+2, 17, color.RGBA{0, 0, 0, 0x50}, true)
			disque(img, x, y-6, 17, coulBlanc, true)
			cercle(img, x, y-6, 17, 2.6, coulEncre, true)
			ecrireCentre(img, fmt.Sprint(v.Valeur), x, y-6-17, fv, coulEncre)
		}
		nom := nomAffiche(v.Nom)
		w := largeur(nom, fn)
		nx := min(max(x-w/2, 6), LargPlat*kPlat-w-6)
		ecrireHalo(img, nom, nx, y+12, fn, coulBlanc, coulEncre, 2.4)
	}
}

func nomAffiche(n string) string {
	r := []rune(n)
	out := make([]rune, 0, len(r))
	for _, c := range r {
		out = append(out, []rune(majuscule(string(c)))...)
	}
	return string(out)
}

func majuscule(s string) string {
	switch s {
	case "é":
		return "É"
	case "è":
		return "È"
	case "ã":
		return "Ã"
	case "ç":
		return "Ç"
	case "î":
		return "Î"
	}
	if len(s) == 1 && s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0] - 32)
	}
	return s
}
