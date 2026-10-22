// Code généré par outils/decoupe ; NE PAS MODIFIER.

package jeu

// Terrain de chaque case, rangée par rangée, de gauche à droite :
// m mer, i île, p plaine, d désert, j jungle, n montagne ; en majuscule, la case porte un fleuve.
var rangeesPlateau = [NbRang]string{
	"mmmmmmmmmmmmmmmmmmmmmmmmmmpp",  // 0
	"imiiimmmmmmmmmmmmmmmmmmmpmppp", // 1
	"pmmmimmmmmmmmmmmmmmmmmmppppp",  // 2
	"ppmmmimmmmmmmmmimmmmmmmpppmpp", // 3
	"pmmmpmmmmmmmmmmmmmmmmmpppmmp",  // 4
	"mppnjmmmmmmmmmmmmmmmmppppppmm", // 5
	"mmnjjjpmmmmmmmmmmmmmpnnnnppp",  // 6
	"mpnjjjjjmmmmmmmmmmimpdddddddm", // 7
	"mnnjjjJjmmmmmmmmmmmpdddddddd",  // 8
	"mnNJJjJjjmmmmmmmmmmpddddddddd", // 9
	"pnnjJJJjjmmmmmmmmmpddddddnnn",  // 10
	"mpnnjJJJJjmmmmmmmmmpddDDdddDd", // 11
	"mmnnJjJJjjdmmmmmimppPppPpddd",  // 12
	"mmmnnnJdjdddpmmmmmmpjjjpJpppp", // 13
	"mmpnnjJdddnnpmmmmmmppjjjJJjp",  // 14
	"mmpnJpJdddnppmmmmmmmmppppjPjj", // 15
	"mpnnPJPDJjpmmmmmmmmmmmmmpPpj",  // 16
	"mpnnPJPdnnmmmmmmmmmmmmmmmmmjJ", // 17
	"pnnPPPpppmmmmmmmmmmmmmmmmmjJ",  // 18
	"pnpPpppmmmmmmmmmimmmmmmmmmpJj", // 19
	"nppPppmmmmmmmmmmmmmmmmmmmmpj",  // 20
	"npppmmmmmmmmmmmmmmmmmmmmmmppj", // 21
	"pPpmmmmmmmmmmmmmmmmmmmmmmppj",  // 22
	"ppmmmmmmmmmmmmmmmmmmmmmmmnpjp", // 23
	"pmmmmmmmmmmmmmmmmmmmmmmmnppp",  // 24
	"pPPmmmmmmmmmmmmmmmmimmmmnnppp", // 25
	"PPmmmmmmmmmmmmmmmmmmmmmpnppp",  // 26
}

// villesPlateau : nom, valeur du sac (×1000 F), case.
var villesPlateau = []Ville{
	{"Paris", 0, Case{27, 1}},
	{"Madrid", 2, Case{24, 2}},
	{"Alger", 3, Case{24, 5}},
	{"Casablanca", 4, Case{22, 5}},
	{"Tunis", 3, Case{27, 6}},
	{"Îles Canaries", 6, Case{18, 7}},
	{"Açores", 7, Case{15, 3}},
	{"Grandes Antilles", 12, Case{3, 1}},
	{"Petites Antilles", 12, Case{5, 3}},
	{"Bogota", 15, Case{3, 7}},
	{"Cayenne", 12, Case{7, 7}},
	{"Lima", 16, Case{1, 9}},
	{"Belém", 11, Case{9, 11}},
	{"La Paz", 14, Case{4, 12}},
	{"Îles du Cap-Vert", 8, Case{16, 12}},
	{"Dakar", 8, Case{18, 12}},
	{"Fort-Lamy", 6, Case{26, 12}},
	{"Abidjan", 8, Case{21, 14}},
	{"Salvador", 18, Case{11, 15}},
	{"Asunción", 15, Case{4, 17}},
	{"Santiago", 18, Case{0, 18}},
	{"São Paulo", 15, Case{6, 18}},
	{"Rio", 14, Case{8, 18}},
	{"Ascension", 12, Case{16, 19}},
	{"Brazzaville", 9, Case{27, 19}},
	{"Buenos Aires", 17, Case{3, 20}},
	{"Sainte-Hélène", 12, Case{19, 25}},
}
