# REST API

Un API public-facing REST est disponible pour ajouter
des commits dans le log distribué.

Le service est hosté sur le port `8080` pour l'instant.
Une CLI sera rajoutée éventuellement pour pouvoir modifier
ces paramètes sans changer le code.

Deux routes sont actuellement disponibles:

- POST: `http://localhost:8080/`

Post ajoute un log à la liste des logs du service. Le service
retournera alors l'offset associé au record posté.

Pour faire une requête, le service s'attend à un JSON de ce format:

```json
{
  "record": {
    "value": "SGVsbG8sIFdvcmxkIQo="
  }
}
```

Le champ `value` peut vous sembler innatendu. Comme nous sauvegardons les
messages en binaire directement sur le disque, nous avons besoin
d'encoder les strings en `base64` avant de les envoyer à l'API.

Exemple:

```bash
http POST localhost:8080 "record[value]=$(echo Hello, World! | base64)"
```

- GET: `http://localhost:8080/`

L'objet retourné sera un JSON contenant le message et son offset. L'offset
est attribué par le service et représente l'index dans la liste des logs.

Get s'attend à un JSON de ce format lors de la requête:

```json
{
  "offset": 0
}
```

Exemple:

```bash
http GET localhost:8080 offset:=0
```
