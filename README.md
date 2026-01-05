# Indiko

The canonical repo for this is hosted on tangled over at [`dunkirk.sh/pipes`](https://tangled.org/@dunkirk.sh/pipes)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/taciturnaxolotl/indiko.git
cd indiko
```

2. Install dependencies:

```bash
bun install
```

3. Create a `.env` file:

```bash
cp .env.example .env
```

Configure the following environment variables:

```env
ORIGIN=https://your-indiko-domain.com
RP_ID=your-indiko-domain.com
PORT=3000
NODE_ENV=production
```

- `ORIGIN` - Full URL where Indiko is hosted (must match RP_ID)
- `RP_ID` - Domain for WebAuthn (no protocol, matches ORIGIN domain)
- `PORT` - Port to run the server on
- `NODE_ENV` - Environment (dev/production)

The database will be automatically created at `./indiko.db` on first run.

4. Start the server:

```bash
# Development (with hot reload)
bun run dev

# Production
bun run start
```

<p align="center">
    <img src="https://raw.githubusercontent.com/taciturnaxolotl/carriage/main/.github/images/line-break.svg" />
</p>

<p align="center">
    <i><code>&copy 2025-present <a href="https://dunkirk.sh">Kieran Klukas</a></code></i>
</p>

<p align="center">
    <a href="https://tangled.org/dunkirk.sh/indiko/blob/main/LICENSE.md"><img src="https://img.shields.io/static/v1.svg?style=for-the-badge&label=License&message=O'Saasy&logoColor=d9e0ee&colorA=363a4f&colorB=b7bdf8"/></a>
</p>
