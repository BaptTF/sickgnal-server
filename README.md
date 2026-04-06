# sickgnal-server

Serveur de relais pour l'application de clavardage chiffre de bout en bout Sickgnal. Le serveur gere l'authentification des utilisateurs, la distribution des cles publiques (X3DH) et le stockage/transfert des messages chiffres. Il ne manipule jamais le contenu des messages en clair.

## Prerequis

- Go 1.25+
- Un compilateur C (gcc, clang, etc.) -- requis par le driver SQLite

## Installation

```bash
git clone https://github.com/BaptTF/sickgnal-server.git
cd sickgnal-server
go mod download
```

## Compilation

```bash
go build -o sickgnal-server .
```

## Utilisation

### Lancer le serveur (TCP)

```bash
./sickgnal-server
```

Le serveur ecoute par defaut sur le port 8080 en TCP non chiffre, avec une base de donnees SQLite dans le repertoire courant.

### Lancer le serveur (TLS)

```bash
./sickgnal-server -tls-cert server.crt -tls-key server.key
```

TLS est active uniquement lorsque les deux options `-tls-cert` et `-tls-key` sont fournies.

### Lancer directement sans compiler

```bash
go run .
```

## Arguments

| Argument    | Type     | Defaut        | Description                                                              |
|-------------|----------|---------------|--------------------------------------------------------------------------|
| `-port`     | `int`    | `8080`        | Port d'ecoute du serveur                                                |
| `-tls-cert` | `string` | _(vide)_      | Chemin vers le fichier de certificat TLS. Active TLS avec `-tls-key`    |
| `-tls-key`  | `string` | _(vide)_      | Chemin vers la cle privee TLS. Active TLS avec `-tls-cert`              |
| `-db`       | `string` | `sickgnal.db` | Chemin vers le fichier de base de donnees SQLite                         |

Exemple avec toutes les options :

```bash
./sickgnal-server -port 9090 -tls-cert /etc/ssl/server.crt -tls-key /etc/ssl/server.key -db /var/lib/sickgnal/data.db
```

## Tests

```bash
go test ./...
```

## Licence

Projet academique -- Universite de Sherbrooke, session hiver 2026.
