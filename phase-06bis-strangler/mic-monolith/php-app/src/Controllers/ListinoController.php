<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class ListinoController extends BaseController
{
    protected function type(): string { return 'listino'; }

    protected function toDto(array $r): array
    {
        return [
            'id'           => $r['id'],
            'codice'       => $r['code'],
            'nome'         => $r['name'],
            'descrizione'  => $r['description'],
            'valido_da'    => $r['date_1'],
            'valido_a'     => $r['date_2'],
            'status'       => $r['status'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'        => $b['codice'] ?? $b['code'] ?? null,
            'name'        => $b['nome']   ?? $b['name'] ?? null,
            'description' => $b['descrizione'] ?? $b['description'] ?? null,
            'date_1'      => $b['valido_da'] ?? null,
            'date_2'      => $b['valido_a']  ?? null,
            'status'      => $b['status'] ?? 'attivo',
        ];
    }

    /** GET /api/listini/:id/voci */
    public function voci(int $id): array
    {
        $rows = $this->repo->rawAll(
            'SELECT bd.* FROM business_data bd
              WHERE bd.record_type = ? AND bd.parent_type = ? AND bd.parent_id = ?
              ORDER BY bd.id LIMIT 500',
            ['voce_listino', 'listino', $id]
        );
        return ['data' => array_map(fn($r) => [
            'id'        => (int)$r['id'],
            'codice'    => $r['code'],
            'nome'      => $r['name'],
            'prezzo'    => $r['amount_1'] !== null ? (float)$r['amount_1'] : null,
            'listino_id'=> (int)$r['parent_id'],
            'status'    => $r['status'],
        ], $rows)];
    }

    /** POST /api/listini/:id/voci */
    public function addVoce(int $id): array
    {
        $body = $this->readJsonBody();
        if (!$body) return [400, ['error' => 'invalid_body']];
        $articolo_id = (int)($body['articolo_id'] ?? 0);
        $prezzo = (float)($body['prezzo'] ?? 0);
        if (!$articolo_id || $prezzo <= 0) {
            return [400, ['error' => 'articolo_id e prezzo richiesti']];
        }
        $art = $this->repo->findById($articolo_id);
        if (!$art || $art->recordType !== 'articolo') {
            return [400, ['error' => 'articolo_id non valido']];
        }
        $listino = $this->repo->findById($id);
        if (!$listino || $listino->recordType !== 'listino') {
            return [404, ['error' => 'listino non trovato']];
        }
        $voce = $this->repo->create('voce_listino', [
            'code'      => $listino->code . '-' . $art->code,
            'name'      => $listino->name . ' - ' . $art->code,
            'parent_id' => $id,
            'parent_type' => 'listino',
            'amount_1'  => $prezzo,
            'status'    => 'attivo',
        ]);
        $this->repo->relate($voce->id, $id, 'voce_di_listino');
        $this->repo->relate($voce->id, $articolo_id, 'articolo_di_voce', $prezzo);
        return [201, ['data' => [
            'id' => $voce->id, 'codice' => $voce->code, 'prezzo' => $prezzo,
            'articolo_id' => $articolo_id, 'listino_id' => $id,
        ]]];
    }
}
