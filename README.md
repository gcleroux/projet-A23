# Projet IFT605

## Auteurs

- Guillaume Cléroux

## Installation

Ce programme fournit un `nix flake` afin de reproduire l'environnement
de développement sans configuration. Le nix flake supporte les processeur
`x86_64` et `ARM64` pour les systèmes d'exploitations Linux, MacOS et
Windows avec WSL2+systemd. Pour reproduire l'environnement de développement, veuillez
faire les commandes suivantes:

```bash
# Plus d'info, https://github.com/DeterminateSystems/nix-installer
curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install
```

```bash
# Entre dans l'environnement de développement
nix develop
```

Alternativement, vous pouvez aussi build from source. Vous aurez besoin des
dépendences suivantes:

- `go >= 1.20`
- `kuberbernetes >= 1.16`
- `protoc`
- `protoc-gen-go`
- `grpc-gateway`
- `mage`
- `httpie`

## Utilisation

_Toutes les commandes suivantes prennent pour acquis que vous êtes dans la
`nix-shell` que nous fournissons._

Une bonne partie de l'utilisation se fait par l'entremise d'un `magefile.go`.
Cela nous permet de produire des exécutables cross-platform.

Pour voir toutes les commandes disponibles, vous pouvez faire la commande suivante
dans le terminal:

```bash
mage
```

### Build

Pour build les exécutables client/serveur, faites la commande suivante:

```bash
mage build
```

Cette commande va créer un executable pour le serveur et le client nommés respectivement
`serveur` et `client`.

### Run

Pour executer le service, il ne suffit que d'utiliser les executables produit à l'étape
précédente:

```bash
# Partir le serveur gRPC
exec server
```

```bash
# Partir le client REST
exec client
```

Le service sera disponible à l'adresse `http://localhost:8080`

Vous pouvez tester l'API avec l'outil `http` disponible dans l'environnement
de développpement. Pour plus d'infos, voir [cette page](./docs/REST.md) dans la docs.

### Test

Nous avons des tests unitaires pour les modules majeurs. Si vous voulez exécuter
les tests, faites la commande suivante:

```bash
mage test
```
