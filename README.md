# metroweb

Protótipo de cliente YouTube Music: backend em Go (só stdlib) + frontend HTML embutido no binário.

## Rodar

```sh
go run .                       # http://localhost:8080
go run . -addr 0.0.0.0:8080    # acessível pelo celular na mesma rede
go build -o metroweb .         # binário único, HTML já embutido
```

Requer `yt-dlp` e `ffmpeg` no PATH. O `yt-dlp` é o primeiro resolvedor de stream e é quem faz os downloads; os clientes innertube ficam de reserva.

### Se o YouTube devolver 403 / só o formato 18

1. Mantenha o yt-dlp atualizado (`pip install -U yt-dlp`): versões com mais de ~90 dias param de funcionar.
2. Instale o provedor de PO token [bgutil-ytdlp-pot-provider](https://github.com/Brainicism/bgutil-ytdlp-pot-provider) (modo script): clone em `~/bgutil-ytdlp-pot-provider`, rode `npm ci && npx tsc` em `server/` e coloque o zip do plugin em `~/.config/yt-dlp/plugins/`.

### Download pra biblioteca (Jellyfin)

```sh
go run . -library ~/Music/metroweb -jellyfin-url http://localhost:8096 -jellyfin-key <API_KEY>
```

Botões ⬇ (baixar) e ♥ (favoritar e baixar). Salva em `Artista/Álbum/Título.m4a` com tags e capa quadrada; `.metroweb-archive.txt` evita duplicatas. Se URL e chave do Jellyfin forem passadas, pede rescan.

## Como funciona

| Rota | O que faz |
|---|---|
| `GET /api/search?q=` | innertube `search` (cliente WEB_REMIX, filtro "Músicas") |
| `GET /api/radio/{id}` | innertube `next` com playlist `RDAMVM{id}`, o rádio do YTM |
| `GET /api/stream/{id}?fmt=opus\|aac` | resolve o áudio e faz proxy com suporte a Range (seek) |

Resolução do stream, em ordem:
1. `player` com cliente **ANDROID_VR**, que devolve URLs diretas sem cifra
2. `player` com cliente **IOS**
3. `yt-dlp -g`, se instalado

As URLs ficam em cache até perto do `expire`. Se o googlevideo responder 403, o cache é limpo e a URL é resolvida de novo uma vez.

O frontend escolhe opus (webm) ou aac (m4a) conforme o navegador suporta.

## Frontend

- Busca com debounce
- Clicar numa música toca agora; ＋ adiciona à fila
- Quando a fila está acabando, o rádio da música atual é anexado automaticamente
- Media Session: teclas de mídia e controles na tela de bloqueio do celular
- Atalhos: espaço = play/pause, shift+←/→ = anterior/próxima, `/` = busca
- Fila e volume salvos no localStorage

## Quando quebrar

O YouTube muda os clientes de vez em quando. O primeiro lugar a mexer é `streamClients` / `webRemix` em `yt.go` (versões e user-agents). O extractor do yt-dlp e o módulo `innertube` do Metrolist são boas referências pros valores atuais.

## Próximos passos possíveis

- Login (cookies do navegador) pra biblioteca, curtidas e playlists
- Letras sincronizadas (LRCLIB)
- Download e cache local
- Virar PWA pra instalar no celular
