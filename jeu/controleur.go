package jeu

// Controleur : les décisions d'un joueur. Le moteur appelle ces méthodes au
// fil des phases ; Equiper et Escale agissent sur l'état par ses méthodes
// (Acheter, DevisPiste, Construire, Entreposer, Reprendre) avant de rendre la main.
type Controleur interface {
	Equiper(e *Etat, j int)                                // phase 3 : achats d'avions et d'aérodromes
	ChoisirAvion(e *Etat, j int, restants []*Avion) *Avion // phase 4 : avion à déplacer (nil : fin)
	Decoller(e *Etat, a *Avion) bool                       // après le jet d'autonomie
	Voler(e *Etat, a *Avion) Ordre                         // case suivante ou atterrissage
	CaseDeCrash(e *Etat, a *Avion, choix []Case) Case      // jet de panne 3
	Escale(e *Etat, a *Avion)                              // posé sur sa propre piste
}
