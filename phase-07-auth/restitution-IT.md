# Restituzione — Fase 07: autenticazione e la prima policy (Gruppo 3)

> English version: [`restitution.md`](./restitution.md)
> Script parlato per la restituzione di CP7 (~4 min). Confrontiamo la *forma*, non la sintassi.

## Apertura
Siamo il Gruppo 3. In CP6bis avevamo il taglio Strangler: letture e creazioni passavano già dal BC. CP7
mette una **busta di identità** attorno all'API del BC. Il taglio è **rotto di proposito**: il Warehouse
BC ora sta dietro un middleware di auth e risponde **401** a chi non porta un JWT valido. La UI di MIC
chiama `/api/articles` **senza token**, quindi gli articoli migrati erano "giù". Tutta la macchina di
identità è **data**: un mock IAM in C# (utenti demo **Alice** e **Bob**) firma token RS512; il BC ne
verifica la firma via JWKS, esegue la checklist dei claim, costruisce un `AuthContext` e specchia
`X-TS-ID`/`X-Workspace-ID` su ogni response. Sotto continua a girare il **dual-write di CP4**. Due cose
mancavano, entrambe nostre.

## Parte 1 — insegnare a MIC a fare login (JavaScript)
Nel `auth-overlay.js` (iniettato dalla facade prima dell'`app.js` di MIC, così il monolite resta
intoccato) abbiamo completato due punti. Il **wrapper di `window.fetch`**: sulle chiamate same-origin
`/api/*` aggiunge `Authorization: Bearer <token da sessionStorage>` e `X-Workspace-ID`, clonando l'`init`
senza mutarlo, e lasciando intatte `/iam/*` ed esterne. E **`loginDemoUser`**: `POST` form-urlencoded a
`/iam/oauth/token` (grant `password`, `client_id` utente, `scope` openid), che salva l'`access_token`.
Questo **ripara il 401**: dopo login la lista carica e le response portano l'identità. Ma Alice e Bob
restano ancora indistinguibili — l'autenticazione dice *chi sei*, non *cosa puoi fare*.

## Parte 2 — la prima policy di autorizzazione (Rego)
La decisione non è in Go: vive in `policies/warehouse.rego`, che il BC **compila dentro** (`go:embed`) e
interroga su ogni rotta protetta. Partiva con `default allow := false` — negava tutto. Abbiamo aggiunto
tre regole: `article:read` per **ogni** chiamante autenticato; `article:create` solo per i principal in
`article_creators` (utenti identificati per **email**, servizi M2M per **service_id**, discriminati da
`principal.kind`); qualsiasi altra azione **negata**. Nota la somiglianza con la checklist di auth: il
buttafuori controllava quale *applicazione* chiedeva il token; la policy controlla cosa *questa identità*
può fare. Stessa forma — una lista e un default — un livello più su.

## Le tre domande: 401, 403, 404
È il filo conduttore della fase. **401** chiede *chi sei?* (nessuna identità). **403** dice *ti conosco,
e no* (identità valida, permesso mancante). **404**, con token valido, dice *sei dentro, non esiste*. Tre
domande diverse — autenticazione, autorizzazione, esistenza — e in questa fase le abbiamo toccate tutte e
tre.

## Come l'abbiamo dimostrato
- **Test** (`docker compose run --build --rm test`): tutti verdi, incluso `TestCan` (7/7), la spec della
  policy che partiva rossa.
- **End-to-end** (facade → adapter → BC → dual-write): senza token `401`; Alice e Bob leggono `200`;
  **Alice crea `201`** (stesso id in **entrambi** gli store — `19.99` DECIMAL nel legacy, `1999` cents nel
  warehouse), **Bob crea `403`**, e il servizio M2M `svc-orders` crea `201`. Le response di Alice portano
  `X-TS-ID` col suo `sub` e `X-Workspace-ID: ws-acme`.
- `git diff` tocca **solo** i due file (`auth-overlay.js`, `warehouse.rego`).

## Le cinque domande (Q1–Q5)
**Q1 — perché partire dal 401 invece di consegnare l'overlay già funzionante?** Perché il 401 rende la
cucitura *visibile*: accendere il confine è un breaking change reale — rompe ogni client che non manda
identità. Prima si osserva il guasto, poi lo si ripara capendo *quale* identità viaggia e perché. Un
overlay già pronto avrebbe nascosto proprio la lezione.

**Q2 — perché propagare il token dell'utente invece di far coniare all'adapter un token M2M per tutto?**
Perché il token utente conserva l'identità reale end-to-end: Alice resta Alice fino al BC, nel `X-TS-ID` e
nell'audit. Un unico token M2M dell'adapter appiattirebbe tutti sullo stesso servizio — si perderebbe *chi*
ha fatto *cosa*, e la policy non potrebbe distinguere Bob da Alice. Evita anche che l'adapter diventi un
*confused deputy* sovra-privilegiato. Token utente = least privilege + accountability.

**Q3 — 401/403/404, quale domanda risponde ciascuno e dove li abbiamo incontrati oggi?** 401 = *chi sei?*
(prima del login, nessun token). 403 = *ti conosco, e no* (Bob prova a creare). 404 = *sei dentro, non
esiste* (token valido su un id inesistente). Autenticazione, autorizzazione, esistenza.

**Q4 — perché deny-by-default è l'unico default sicuro, e dove l'avevamo già incontrato?** Perché ogni
caso non previsto (una nuova azione, un typo, un principal dimenticato) **fallisce chiuso**, non aperto:
un buco dimenticato nega, non concede. L'idea era già lì due volte: l'allowlist di auth (il buttafuori
ammette solo le applicazioni in lista) e `policy.go` che **fallisce chiuso** sugli errori di valutazione —
e che va addirittura in panic se la policy non compila, invece di negare in silenzio.

**Q5 — le regole sono già Rego, il linguaggio del motore di policy della Fase 08: cosa resta da cambiare?**
Non il linguaggio né le regole: la loro **casa**. Oggi `warehouse.rego` è compilato dentro il BC —
cambiare una regola (aggiungere Bob ai creatori, una nuova policy per tenant) richiede una **ricompilazione
e un redeploy** del BC. In Fase 08 lo stesso Rego è servito da un motore esterno (Policy Manager):
versionato, auditabile e modificabile **senza ricompilare** il BC. Cambia il *ciclo di vita* della policy,
non le regole.

## Come si esegue
```bash
cd phase-07-auth
docker compose -f mic-integration/docker-compose.auth.yml up -d --build   # BC (Go) + IAM mock (C#) + MIC + facade
docker compose -f mic-integration/docker-compose.auth.yml --profile tools run --rm backfill-articles
docker compose run --build --rm test                                       # suite verde (incl. TestCan)
#  UI: http://localhost:8081/#/articles  ->  Login Alice/Bob  ->  crea (201 / 403)
docker compose -f mic-integration/docker-compose.auth.yml down
```

## Chiusura
Il BC estratto ora ha una busta di identità: un JWT valido apre il confine (autenticazione), e una policy
Rego decide cosa quell'identità può fare (autorizzazione), con la persistenza dual-write di CP4 ancora
sotto. Le stesse regole, in Fase 08, lasceranno il binario per un motore di policy esterno. Domande?

## Glossario
- **Autenticazione vs autorizzazione** — *chi sei* (JWT valido, 401 se no) vs *cosa puoi fare* (policy, 403 se no).
- **JWKS** — il set di chiavi pubbliche con cui il BC verifica la firma dei token senza conoscere la chiave privata dell'IAM.
- **AuthContext** — l'identità estratta dal token (kind utente/M2M, email, service_id) su cui decide la policy.
- **Rego / OPA** — il linguaggio dichiarativo di policy; il BC lo valuta in-process e legge `allow`.
- **Deny-by-default** — tutto ciò che non è esplicitamente permesso è negato: i casi dimenticati falliscono chiusi.
- **Token utente vs M2M** — l'utente porta la propria identità end-to-end; l'M2M identifica un servizio (`svc-orders`).
