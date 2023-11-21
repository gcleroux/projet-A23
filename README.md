# Projet IFT605

## Auteurs

- Guillaume Cléroux

## Installation

Ce programme fournit un `nix flake` afin de reproduire l'environnement
de développement sans configuration. Le nix flake supporte Linux, MacOS et
Windows avec WSL2. Pour reproduire l'environnement de développement, veuillez
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

- `go >= 1.13`
- `kuberbernetes >= 1.16`
- `protoc`
- `protoc-gen-go`
- `mage` (non-nécessaire, mais très pratique)

## Utilisation

### Build

Pour build le service, nous avons un fichier `magefile.go` pour permettre un build
cross-platform. À partir de la `nix-shell`, faites:

```bash
mage build
```

Cette commande va créer un executable pour votre plateforme dans le dossier `bin/`.

### Run

Pour executer le service, il ne suffit que d'utiliser l'exectuable produit à l'étape
précédente:

```bash
exec ./bin/main
```

Le service sera disponible à l'adresse `http://localhost:8080`

Vous pouvez tester l'API avec l'outil `http` disponible dans l'environnement
de développpement. Pour plus d'infos, voir [cette page](./docs/REST.md) dans la docs.

### Test

Nous avons des tests pour le module `log`, si vous voulez exécuter les tests,
faites la commande suivante:

```bash
mage test
```
