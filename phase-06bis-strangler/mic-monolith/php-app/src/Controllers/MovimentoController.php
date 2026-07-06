<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class MovimentoController extends BaseController
{
    protected function type(): string { return 'movimento'; }

    protected function toDto(array $r): array
    {
        $articolo = null; $magazzino = null;
        if (!empty($r['id'])) {
            $articolo = $this->lookupRelTarget((int)$r['id'], 'movimento_di_articolo');
            $magazzino = $this->lookupRelTarget((int)$r['id'], 'movimento_in_magazzino');
        }
        return [
            'id'          => $r['id'],
            'codice'      => $r['code'],
            'nome'        => $r['name'],
            'data'        => $r['date_1'],
            'qta'         => $r['amount_1'],
            'tipo'        => $r['text_1'],
            'causale'     => $r['payload']['causale'] ?? null,
            'articolo_id' => $articolo['id'] ?? null,
            'articolo_sku'=> $articolo['code'] ?? null,
            'magazzino_id'=> $magazzino['id'] ?? null,
            'magazzino_nome'=> $magazzino['name'] ?? null,
            'status'      => $r['status'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'      => $b['codice'] ?? $b['code'] ?? ('MOV-'.uniqid()),
            'name'      => $b['nome']   ?? ($b['tipo'] ?? 'movimento'),
            'date_1'    => $b['data']   ?? date('Y-m-d'),
            'amount_1'  => $b['qta']    ?? 0,
            'text_1'    => $b['tipo']   ?? 'entrata',
            'status'    => $b['status'] ?? 'confermato',
            'payload'   => ['causale' => $b['causale'] ?? null],
        ];
    }

    private function lookupRelTarget(int $sourceId, string $type): ?array
    {
        $row = $this->repo->rawOne(
            'SELECT bd.id, bd.code, bd.name FROM business_relations br
              JOIN business_data bd ON bd.id = br.target_id
             WHERE br.source_id = ? AND br.relation_type = ?
             LIMIT 1',
            [$sourceId, $type]
        );
        return $row ?: null;
    }

    /** Override create to also wire the relations to articolo and magazzino. */
    public function create(): array
    {
        $body = $this->readJsonBody();
        if (!$body) return [400, ['error' => 'invalid_body']];
        $art_id = (int)($body['articolo_id'] ?? 0);
        $mag_id = (int)($body['magazzino_id'] ?? 0);
        if (!$art_id || !$mag_id) {
            return [400, ['error' => 'articolo_id e magazzino_id richiesti']];
        }
        $b = $this->repo->create('movimento', $this->fromPayload($body));
        $this->repo->relate($b->id, $art_id, 'movimento_di_articolo', (float)($body['qta'] ?? 0));
        $this->repo->relate($b->id, $mag_id, 'movimento_in_magazzino');
        return [201, ['data' => $this->toDto($b->toArray())]];
    }
}
