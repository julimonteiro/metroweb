# Handoff: download pra biblioteca do Jellyfin

Arquivo pra continuar no Claude Code. Pode apagar quando terminar.

## Já feito (compila, `go vet` e `go test` passam)

- `library.go`: fila de downloads com 1 worker usando `yt-dlp` + `ffmpeg`
  - Saída em `<library>/Artista/Álbum/NN - Título.m4a`, com metadados e capa embutidos (a capa é cortada pra ficar quadrada)
  - Arquivo `.metroweb-archive.txt` na pasta da biblioteca evita baixar a mesma música duas vezes e marca como "done" quando o servidor reinicia
  - Rescan do Jellyfin (`POST /Library/Refresh`) com debounce de 15s, se URL e chave estiverem configuradas
- `main.go`: flags `-library` (padrão `~/Music/metroweb`), `-jellyfin-url` e `-jellyfin-key`, ou as env vars `METROWEB_LIBRARY`, `JELLYFIN_URL` e `JELLYFIN_API_KEY`
  - `POST /api/download/{id}` devolve `{status}`, ou 503 se faltar yt-dlp/ffmpeg
  - `GET /api/downloads` devolve um mapa `id -> queued|running|done|error: ...`

## Falta

1. **Frontend (`web/index.html`):** botão ⬇ em cada linha, nos resultados e na fila, e no player (#now)
   - Estados: ⬇ (nada), ⏳ (queued/running), ✓ (done), ⚠ (error, com a mensagem no `title`)
   - Ao clicar: `POST /api/download/{id}`
   - Carregar `/api/downloads` no boot e fazer polling a cada 2s só enquanto tiver algum queued/running
   - Se der 503, mostrar a mensagem no `#status`
2. **Testar de verdade no Mac:** `brew install yt-dlp ffmpeg`, depois `go run . -library "<pasta do Jellyfin>"`
   - Busca, play, rádio e download
   - Conferir se o Jellyfin reconhece artista, álbum e capa
3. Se o stream não resolver pelos clientes ANDROID_VR/IOS, atualizar as versões em `yt.go` (usar o yt-dlp como referência)

## Prompt sugerido

> Lê o HANDOFF.md e o README.md do metroweb, implementa o botão de download no frontend conforme o item 1, depois roda o servidor e testa busca, play e download de verdade. Minha pasta de música do Jellyfin é: ___
