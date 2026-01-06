# Pipes

This is my interperitation of yahoo pipes from back in the day! It is designed to allow you to string together pipelines of data and do cool stuff!

The canonical repo for this is hosted on tangled over at [`dunkirk.sh/pipes`](https://tangled.org/@dunkirk.sh/pipes)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/taciturnaxolotl/pipes.git
cd pipes
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
ORIGIN=https://pipes.yourdomain.com
PORT=3000
NODE_ENV=production
DATABASE_URL=data/pipes.db

# Indiko OAuth Configuration
INDIKO_CLIENT_ID=ikc_xxxxxxxxxxxxxxxxxxxxx
INDIKO_CLIENT_SECRET=iks_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
INDIKO_ORIGIN=https://indiko.dunkirk.sh
INDIKO_REDIRECT_URI=https://pipes.yourdomain.com/auth/callback
```

The database will be automatically created at `./data/pipes.db` on first run.

4. Set up Indiko OAuth:
   - Go to your Indiko instance
   - Navigate to Admin → OAuth Clients
   - Create a new client with the redirect URI matching your `INDIKO_REDIRECT_URI`
   - Copy the Client ID and Secret to your `.env` file

5. Start the server:

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
