<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class InvoiceController extends BaseController
{
    protected function type(): string { return 'fattura'; }

    public function dto(array $r): array { return $this->toDto($r); }

    protected function toDto(array $r): array
    {
        $cliente = null; $ordine = null;
        if (!empty($r['id'])) {
            $cliente = $this->lookupRelTarget((int)$r['id'], 'cliente_di_fattura');
            $ordine  = $this->lookupRelTarget((int)$r['id'], 'fattura_di_ordine');
        }
        return [
            'id'             => $r['id'],
            'numero'         => $r['code'],
            'data'           => $r['date_1'],
            'scadenza'       => $r['date_2'],
            'imponibile'     => $r['amount_1'],
            'iva'            => $r['amount_2'],
            'totale'         => $r['amount_3'],
            'sezionale'      => $r['text_1'],
            'anno'           => $r['text_2'],
            'cliente_id'     => $cliente['id']   ?? null,
            'cliente_nome'   => $cliente['name'] ?? null,
            'ordine_id'      => $ordine['id']    ?? null,
            'ordine_numero'  => $ordine['code']  ?? null,
            'sdi_status'     => $r['payload']['sdi_status'] ?? null,
            'status'         => $r['status'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'      => $b['numero'] ?? $b['code'] ?? null,
            'name'      => 'Fattura ' . ($b['numero'] ?? ''),
            'date_1'    => $b['data'] ?? date('Y-m-d'),
            'date_2'    => $b['scadenza'] ?? date('Y-m-d', strtotime('+30 days')),
            'amount_1'  => $b['imponibile'] ?? 0,
            'amount_2'  => $b['iva']        ?? 0,
            'amount_3'  => $b['totale']     ?? 0,
            'text_1'    => $b['sezionale']  ?? 'VEN',
            'text_2'    => $b['anno']       ?? date('Y'),
            'status'    => $b['status']     ?? 'bozza',
            'payload'   => ['sdi_status' => $b['sdi_status'] ?? 'DRAFT'],
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

    public function create(): array
    {
        $body = $this->readJsonBody();
        if (!$body) return [400, ['error' => 'invalid_body']];
        $cliente_id = (int)($body['cliente_id'] ?? 0);
        $ordine_id  = (int)($body['ordine_id'] ?? 0);
        if (!$cliente_id) return [400, ['error' => 'cliente_id richiesto']];

        $b = $this->repo->create('fattura', $this->fromPayload($body));
        $this->repo->relate($b->id, $cliente_id, 'cliente_di_fattura');
        if ($ordine_id) $this->repo->relate($b->id, $ordine_id, 'fattura_di_ordine');
        return [201, ['data' => $this->toDto($b->toArray())]];
    }

    /** POST /api/invoices/:id/invia-sdi (mock) */
    public function inviaSdi(int $id): array
    {
        $b = $this->repo->findById($id);
        if (!$b || $b->recordType !== 'fattura') return [404, ['error' => 'not_found']];
        $payload = $b->payload ?? [];
        $payload['sdi_status'] = 'INVIATA';
        $payload['sdi_at']     = date('c');
        $this->repo->update($id, ['status' => 'inviato', 'payload' => $payload]);
        return ['data' => ['id' => $id, 'sdi_status' => 'INVIATA']];
    }
}
