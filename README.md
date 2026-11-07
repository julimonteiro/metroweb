# metroweb

Protótipo de cliente YouTube Music: backend em Go (só stdlib) + frontend HTML embutido no binário.

## Rodar

```sh
go run .                       # http://localhost:8080
go run . -addr 0.0.0.0:8080    # acessível pelo celular na mesma rede
go build -o metroweb .         # binário único, HTML já embutido
```

Opcional, mas recomendado: ter `yt-dlp` no PATH (`brew install yt-dlp`). Se os clientes innertube pararem de devolver stream, ele é usado como fallback.

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
