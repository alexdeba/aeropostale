# Aéropostale

Adaptation en Go (Ebitengine) du jeu de François Nédélec paru en encart dans
*Jeux & Stratégie* (1983, dessin Claude Lacroix), d'après la règle et
l'encart du magazine. Vous
dirigez une compagnie d'aviation postale contre une à trois compagnies menées
par l'ordinateur (Mermoz, Guillaumet, Saint-Exupéry) : le premier qui a
rapporté 13 sacs de courrier à Paris gagne.

```
go run .
```

## Déroulement

À l'accueil, choisissez votre couleur et le nombre de compagnies rivales. Chaque
tour : le marqueur T avance, le dé fixe le vent (la flèche rouge de la rose),
puis chaque compagnie joue équipement, déplacements, ravitaillement.

- **Équipement** : achat d'avions, ou d'une piste privée (cliquez la case, le dé
  fixe le devis, vous construisez ou renoncez).
- **Déplacements** : choisissez un avion (cerclé d'or) ; le jet d'autonomie est
  lancé. La zone accessible s'éclaire : sans risque en clair, au-delà de
  l'autonomie (jet de panne à chaque case) en orange ; les escales possibles
  sont cerclées de blanc. Survolez une case pour voir le trajet et son coût,
  cliquez pour décoller et voler. À chaque escale, un nouveau jet : redécollez
  ou restez au sol.
- Le télégramme du panneau donne la consigne de chaque étape ; le bandeau du
  plateau montre le dernier évènement et ses dés ; la feuille de route de
  l'avion (tableau 1 de la règle) se remplit au fil des escales.

Touches : R la règle (6 pages, flèches), A accélérer l'ordinateur, Échap / clic
droit annuler, F11 plein écran.

## Organisation

- `jeu/` : les règles, sans interface — `plateau.go` (27 rangées d'hexagones
  pointe en haut, voisins, distances, rose des vents), `plateau_donnees.go`
  (terrains, fleuves et villes, généré), `etat.go` (avions, pistes, achats,
  courrier, primes, ravitaillement), `vol.go` (tableau 2 : jet d'autonomie,
  coût selon le vent et la montagne, pannes, crash, survie), `tour.go`,
  `ia.go`, `controleur.go`.
- `partie.go` : le moteur joue dans sa goroutine ; le joueur humain répond par un canal.
- `plateau.go`, `jetons.go`, `vue.go`, `panneau.go`, `regles.go`,
  `accueil.go` : interface. Le plateau est entièrement redessiné (textures de
  plaine, désert, jungle, montagne, îlots, fleuves, côtes, villes, rose des
  vents d'Éole, bande des tours) ; les avions sont dessinés vus de dessus, aux
  couleurs des pions, et tournés dans le sens du vol.
- Habillage « poste aérienne des années 1930 » : plateau en timbre-poste
  dentelé, panneau à liseré rouge et bleu des enveloppes *par avion*, consignes
  en télégramme tapé à la machine, boutons Art déco marine et or. Polices
  Limelight, Josefin Sans, Special Elite (OFL / Apache, `assets/polices`).
- Images conservées : la couverture (à l'accueil), le titre
  de la règle (masque d'encre, sur le plateau et le panneau), les trois avions
  dessinés dans la règle (boutons d'achat).
- Le plateau (terrains, fleuves, villes) a été relevé sur une photo de
  l'encart par un outil qui n'est pas distribué : le résultat est dans
  `jeu/plateau_donnees.go`.

## Choix d'adaptation

La règle est appliquée telle quelle ; là où elle est muette ou ambiguë :

- Le numéro du vent (1 à 6, rose de l'encart) est la direction **vers**
  laquelle il souffle : voler dans ce sens, c'est le vent arrière (½ point).
- Le jet de panne a lieu quand l'avion veut quitter une case alors que son
  autonomie est épuisée : on peut encore se poser sur la ville atteinte.
  Un avion ne peut pas se reposer sur sa case de décollage sans avoir bougé
  (ce serait relancer l'autonomie pour un point).
- Une case ne coûte que si l'avion a le carburant pour y entrer ; en vol sans
  case accessible ni escale, c'est la panne sèche (crash).
- Pannes 2 et 3 et panne sèche : l'avion est détruit (l'hydravion sur un fleuve
  reste intact sur 3 à 6), le jet de survie décide du pilote ; la mer, absente
  du barème, compte comme une île (il faut un 6). Panne 4 : l'avion reste au
  sol le tour suivant et repart le tour d'après.
- Un avion prend au plus un sac par ville et par tour ; une piste privée
  (de n'importe qui) voisine d'une ville en donne le courrier. Paris ne fournit
  pas de courrier, et seule l'escale à Paris même livre les sacs.
- La prime de première liaison double la valeur d'un seul sac ; elle est
  réglée à la fin du tour (pour départager deux livraisons du même tour, le
  courrier pris le plus tard l'emporte, comme le dit la règle). L'exemple du
  tableau 1 (25 000 F, prime des Canaries seule) est ainsi respecté.
- Une seule tentative de construction de piste par tour (sinon on relancerait
  le dé du devis à volonté).
- Tout avion acheté part de Paris le plein fait ; un pilote vétéran rescapé
  attend à Paris et prend le prochain avion acheté.
- Le plein est obligatoire en fin de déplacement s'il est possible et payable ;
  sinon l'avion garde ce qui lui reste. L'hydravion posé sur un fleuve ne se
  ravitaille pas.
- Le pion T reprend la bande au début après le tour 20. Garde-fou : au tour 99,
  ou si plus personne ne peut voler, le plus de sacs rapportés l'emporte.
- Pas d'alliances ni de commerce d'avions entre joueurs contre l'ordinateur.
- Les potentiels de pions de l'encart : 6 moyen-courriers, 5 long-courriers,
  4 hydravions et 7 aérodromes par couleur.

## L'ordinateur

Chaque avion suit une mission (une ville cible, puis le retour à Paris avec
le courrier). Après chaque jet d'autonomie, il choisit l'escale accessible sans
entrer en zone de panne qui rapporte le plus (sac à prendre, progression vers
le but sur le réseau des escales, prix du plein), ou reste au sol. Il achète
des moyen-courriers pour la navette, puis des long-courriers, et quelques
pistes relais une fois sa flotte faite. Les parties entre ordinateurs durent
une vingtaine de tours.

## Tests et captures

```
go test ./...
UNE=graine,joueurs go test ./jeu -run TestUne -v      # journal d'une partie
./aeropostale -shot accueil.png                        # accueil
./aeropostale -ecran jeu -graine 4 -tours 3 -arret decoller -survol 24,2 -shot vol.png
./aeropostale -ecran jeu -graine 7 -tours 6 -arret equiper -piste -survol 23,5 -shot piste.png
./aeropostale -ecran jeu -graine 3 -tours 2 -arret equiper -devis 23,3 -shot devis.png
./aeropostale -ecran jeu -graine 2 -temeraire -arret crash -shot panne.png
./aeropostale -ecran jeu -graine 3 -pistedepart 23,3 -arret escale -shot escale.png
./aeropostale -ecran jeu -graine 3 -tours 60 -shot fin.png
./aeropostale -regle 3 -shot regle.png
```

`-arret` fait jouer l'ordinateur à votre place jusqu'à la demande voulue
(equiper, devis, avion, decoller, voler, crash, escale) ; `-temeraire` le fait
voler au-delà de son autonomie pour provoquer des pannes.

## Crédits et licence

*Aéropostale* est un jeu de François Nédélec, paru en encart dans
*Jeux & Stratégie* en 1983, avec des illustrations de Claude Lacroix. Le jeu
original et ses illustrations appartiennent à leurs auteurs. Cette adaptation
est un hommage non commercial.

Les scans de la règle, de l'encart et de la couverture ne sont pas distribués
dans ce dépôt. Seules les images dont le jeu a besoin le sont
(`assets/gen`).

Le code est sous licence MIT (voir `LICENSE`). Les polices de
`assets/polices` gardent leurs propres licences : SIL OFL pour Limelight et
Josefin Sans, Apache 2.0 pour Special Elite.
