# metroweb

Cliente web de YouTube Music, no estilo do Metrolist, feito pra celular e PC. Você busca, ouve, favorita (♥) e **baixa pra uma pasta que o Jellyfin monitora**. Backend em Go (só stdlib) com o HTML embutido no binário; login próprio, pra você hospedar em casa ou num servidor e acessar de qualquer lugar.

- Interface mobile-first (barra de navegação, mini player, player em tela cheia com cor da capa) e layout de desktop com barra lateral e player fixo
- Instalável como PWA no celular
- ♥ favorita **e** baixa a música; ⋮ abre o menu (rádio, tocar a seguir, fila, baixar)
- Download em `Artista/Álbum/Título.m4a`, com tags e capa quadrada; não baixa a mesma música duas vezes; pede rescan ao Jellyfin se você configurar

## Rodar com Docker (recomendado)

```sh
git clone https://github.com/julimonteiro/metroweb && cd metroweb
# edite docker-compose.yml: ./music -> a pasta que o Jellyfin monitora
docker compose up -d --build
```

Abra `http://localhost:8080`. O login inicial é **`admin` / `admin`**: troque pela própria interface (ícone da conta, canto superior direito no celular, "admin" no menu lateral no PC). Enquanto a senha padrão estiver em uso, o app mostra um aviso.

A imagem já traz `yt-dlp`, `ffmpeg` e Node.

## Rodar sem Docker

Precisa de Go 1.24+, `yt-dlp` e `ffmpeg` no PATH.

```sh
go run . -library ~/Music/metroweb -addr 0.0.0.0:8080
```

| Flag | Env | O que faz |
|---|---|---|
| `-addr` | | Endereço de escuta (padrão `:8080`) |
| `-library` | `METROWEB_LIBRARY` | Pasta onde as músicas são salvas (a do Jellyfin) |
| `-jellyfin-url` | `JELLYFIN_URL` | Ex.: `http://jellyfin:8096`; ativa o rescan automático |
| `-jellyfin-key` | `JELLYFIN_API_KEY` | Chave de API (Painel > Chaves de API) |
| `-user` / `-password` | `METROWEB_USER` / `METROWEB_PASSWORD` | Credenciais **iniciais** (só valem na primeira execução; padrão `admin`/`admin`) |
| `-data` | `METROWEB_DATA` | Pasta de dados: credenciais (com hash) e segredo da sessão |
| `-trust-proxy` | `METROWEB_TRUST_PROXY=1` | Usa `X-Forwarded-For` (atrás de proxy/túnel) pro limite de tentativas |

## Acessar de fora de casa

O servidor tem login (senha com PBKDF2, cookie de sessão assinado e `HttpOnly`, bloqueio de 5 minutos após 5 tentativas erradas por IP), mas **fale com ele sempre por HTTPS** quando for pela internet, senão a senha trafega em texto puro. Três jeitos:

1. **Tailscale** (mais simples e mais seguro): instale no servidor e no celular; acesse por `http://nome-do-servidor:8080` sem expor nada na internet. `tailscale serve` dá HTTPS de graça.
2. **Cloudflare Tunnel**: `cloudflared tunnel --url http://localhost:8080` (ou um túnel nomeado com seu domínio). Já vem com HTTPS.
3. **Proxy reverso próprio** (Caddy, nginx, Traefik) com certificado Let's Encrypt, apontando pra porta 8080. Exemplo com Caddy:

   ```
   musica.seudominio.com {
       reverse_proxy localhost:8080
   }
   ```

**Troque a senha `admin` antes de expor na internet.** Qualquer pessoa logada pode enfileirar downloads no seu disco.

## Quando o YouTube der 403 / só aparecer o formato 18

Quase sempre é o `yt-dlp` desatualizado: versões com mais de ~90 dias param de funcionar. Atualize (`pip install -U "yt-dlp[default]"`; no Docker, `docker compose build --no-cache`) e tenha um runtime JS (Deno ou Node) no PATH. Se ainda falhar, o plugin [bgutil-ytdlp-pot-provider](https://github.com/Brainicism/bgutil-ytdlp-pot-provider) gera o PO token que o YouTube às vezes exige.

O `yt-dlp` é o primeiro resolvedor de stream e quem faz os downloads; os clientes innertube em `yt.go` ficam de reserva.

## Como funciona

| Rota | O que faz |
|---|---|
| `GET /api/search?q=` | innertube `search` (cliente WEB_REMIX, filtro "Músicas") |
| `GET /api/radio/{id}` | innertube `next` com playlist `RDAMVM{id}`, o rádio do YTM |
| `GET /api/stream/{id}?fmt=opus\|aac` | resolve o áudio e faz proxy com suporte a Range (seek) |
| `POST /api/download/{id}` | enfileira o download pra biblioteca |
| `GET /api/downloads` | estado de cada download: `queued`, `running`, `done`, `error: …` |
| `GET /api/img?u=` | proxy das capas (só hosts do Google/YouTube) |
| `GET/POST /api/account` | usuário atual / troca de usuário e senha |

Todas, menos `/login` e os arquivos do PWA, exigem sessão.

## Atalhos (PC)

Espaço play/pause · Shift+←/→ anterior/próxima · `/` busca · Esc fecha player/menus.

## Licença e avisos

Projeto pessoal, sem afiliação com YouTube, Google ou Jellyfin. Baixar conteúdo do YouTube pode violar os termos do serviço e a lei de direitos autorais do seu país: use só com o que você tem direito de baixar.
