<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class OrderController extends BaseController
{
    protected function type(): string { return 'ordine'; }

    public function dto(array $r): array { return $this->toDto($r); }

    protected function toDto(array $r): array
    {
        $cliente = null; $listino = null; $agente = null;
        if (!empty($r['id'])) {
            $cliente  = $this->lookupRelTarget((int)$r['id'], 'cliente_di_ordine');
            $listino  = $this->lookupRelTarget((int)$r['id'], 'listino_di_ordine');
            $agente   = $this->lookupRelTarget((int)$r['id'], 'agente_di_ordine');
        }
        return [
            'id'              => $r['id'],
            'numero'          => $r['code'],
            'data'            => $r['date_1'],
            'imponibile'      => $r['amount_1'],
            'iva'             => $r['amount_2'],
            'totale'          => $r['amount_3'],
            'sconto_globale'  => $r['amount_4'],
            'cliente_id'      => $cliente['id']   ?? null,
            'cliente_nome'    => $cliente['name'] ?? null,
            'listino_id'      => $listino['id']   ?? null,
            'listino_nome'    => $listino['name'] ?? null,
            'agente_id'       => $agente['id']    ?? null,
            'agente_nome'     => $agente['name']  ?? null,
            'status'          => $r['status'],
            'note'            => $r['payload']['note'] ?? null,
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'     => $b['numero'] ?? $b['code'] ?? ('ORD-'.date('Y').'-'.str_pad((string)random_int(1,99999), 5, '0', STR_PAD_LEFT)),
            'name'     => 'Ordine ' . ($b['numero'] ?? ''),
            'date_1'   => $b['data'] ?? date('Y-m-d'),
            'amount_1' => $b['imponibile']     ?? 0,
            'amount_2' => $b['iva']            ?? 0,
            'amount_3' => $b['totale']         ?? 0,
            'amount_4' => $b['sconto_globale'] ?? 0,
            'status'   => $b['status'] ?? 'bozza',
            'payload'  => ['note' => $b['note'] ?? null],
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

    /** Override create to also wire cliente/listino/agente relations. */
    public function create(): array
    {
        $body = $this->readJsonBody();
        if (!$body) return [400, ['error' => 'invalid_body']];

        $cliente_id = (int)($body['cliente_id'] ?? 0);
        if (!$cliente_id) return [400, ['error' => 'cliente_id richiesto']];

        $b = $this->repo->create('ordine', $this->fromPayload($body));
        $this->repo->relate($b->id, $cliente_id, 'cliente_di_ordine');
        if (!empty($body['listino_id']))
            $this->repo->relate($b->id, (int)$body['listino_id'], 'listino_di_ordine');
        if (!empty($body['agente_id']))
            $this->repo->relate($b->id, (int)$body['agente_id'], 'agente_di_ordine');
        return [201, ['data' => $this->toDto($b->toArray())]];
    }

    /** GET /api/orders/:id/righe */
    public function righe(int $id): array
    {
        $rows = $this->repo->rawAll(
            'SELECT bd.* FROM business_data bd
              WHERE bd.record_type = ? AND bd.parent_type = ? AND bd.parent_id = ?
              ORDER BY bd.id',
            ['riga_ordine', 'ordine', $id]
        );
        $data = [];
        foreach ($rows as $r) {
            $art = $this->repo->rawOne(
                'SELECT bd.id, bd.code, bd.name
                   FROM business_relations br JOIN business_data bd ON bd.id=br.target_id
                  WHERE br.source_id = ? AND br.relation_type = ? LIMIT 1',
                [(int)$r['id'], 'articolo_in_riga_ordine']
            );
            $payload = $r['payload_json'] ? json_decode($r['payload_json'], true) : [];
            $data[] = [
                'id'              => (int)$r['id'],
                'codice'          => $r['code'],
                'articolo_id'     => $art['id'] ?? null,
                'articolo_nome'   => $art['name'] ?? null,
                'sku'             => $payload['sku'] ?? ($art['code'] ?? null),
                'qta'             => (float)$r['amount_1'],
                'prezzo_unitario' => (float)$r['amount_2'],
                'sconto_riga'     => (float)$r['amount_3'],
                'totale'          => (float)$r['amount_4'],
                'imponibile'      => $payload['imponibile'] ?? null,
                'iva'             => $payload['iva']        ?? null,
                'iva_perc'        => $payload['iva_perc']   ?? null,
            ];
        }
        return ['data' => $data];
    }

    /** POST /api/orders/:id/righe */
    public function addRiga(int $id): array
    {
        $body = $this->readJsonBody();
        if (!$body) return [400, ['error' => 'invalid_body']];
        $art_id = (int)($body['articolo_id'] ?? 0);
        $qta = (float)($body['qta'] ?? 0);
        $prezzo = (float)($body['prezzo_unitario'] ?? 0);
        $sconto = (float)($body['sconto_riga'] ?? 0);
        if (!$art_id || $qta <= 0) return [400, ['error' => 'articolo_id e qta richiesti']];

        $articolo = $this->repo->findById($art_id);
        if (!$articolo) return [400, ['error' => 'articolo non trovato']];
        if ($prezzo <= 0) $prezzo = (float)($articolo->amount1 ?? 0);

        $imp = round($qta * $prezzo * (1 - $sconto/100), 2);
        // IVA dell'articolo (text_2 = codice IVA, lookup percentuale)
        $iva_code = $articolo->text2 ?? 'IVA22';
        $ivaRow = $this->repo->rawOne(
            'SELECT id, amount_1 FROM business_data WHERE record_type=? AND code=? LIMIT 1',
            ['aliquota_iva', $iva_code]
        );
        $iva_perc = $ivaRow ? (float)$ivaRow['amount_1'] : 22.0;
        $iva = round($imp * $iva_perc / 100, 2);
        $tot = round($imp + $iva, 2);

        $righe_count = (int)($this->repo->rawOne(
            'SELECT COUNT(*) c FROM business_data WHERE record_type=? AND parent_type=? AND parent_id=?',
            ['riga_ordine','ordine',$id]
        )['c'] ?? 0);

        $ordine = $this->repo->findById($id);
        $numero = $ordine ? $ordine->code : ('ORD-'.$id);

        $riga = $this->repo->create('riga_ordine', [
            'code'        => sprintf('%s-R%03d', $numero, $righe_count + 1),
            'name'        => $articolo->code,
            'parent_id'   => $id,
            'parent_type' => 'ordine',
            'amount_1'    => $qta,
            'amount_2'    => $prezzo,
            'amount_3'    => $sconto,
            'amount_4'    => $tot,
            'status'      => 'attivo',
            'payload'     => ['imponibile' => $imp, 'iva' => $iva, 'iva_perc' => $iva_perc, 'sku' => $articolo->code],
        ]);
        $this->repo->relate($riga->id, $art_id, 'articolo_in_riga_ordine', $qta);
        if ($ivaRow) $this->repo->relate($riga->id, (int)$ivaRow['id'], 'iva_di_riga', $iva_perc);

        // ricalcola totali ordine
        $this->recalcOrdine($id);
        return [201, ['data' => [
            'id' => $riga->id, 'qta' => $qta, 'prezzo_unitario' => $prezzo,
            'sconto_riga' => $sconto, 'imponibile' => $imp, 'iva' => $iva,
            'totale' => $tot,
        ]]];
    }

    private function recalcOrdine(int $id): void
    {
        $rows = $this->repo->rawAll(
            'SELECT amount_1 qta, amount_2 prezzo, amount_3 sconto, payload_json
              FROM business_data WHERE record_type=? AND parent_type=? AND parent_id=?',
            ['riga_ordine', 'ordine', $id]
        );
        $imp = 0; $iva = 0;
        foreach ($rows as $r) {
            $p = $r['payload_json'] ? json_decode($r['payload_json'], true) : [];
            $imp += (float)($p['imponibile'] ?? 0);
            $iva += (float)($p['iva']        ?? 0);
        }
        $ord = $this->repo->findById($id);
        $sg  = (float)($ord->amount4 ?? 0);
        $imp_eff = round($imp * (1 - $sg/100), 2);
        $iva_eff = round($iva * (1 - $sg/100), 2);
        $tot     = round($imp_eff + $iva_eff, 2);
        $this->repo->update($id, [
            'amount_1' => $imp_eff, 'amount_2' => $iva_eff, 'amount_3' => $tot,
        ]);
    }
}
