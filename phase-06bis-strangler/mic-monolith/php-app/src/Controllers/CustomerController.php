<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class CustomerController extends BaseController
{
    protected function type(): string { return 'cliente'; }

    protected function toDto(array $r): array
    {
        return [
            'id'           => $r['id'],
            'piva_or_cf'   => $r['code'],
            'ragione_sociale' => $r['name'],
            'email'        => $r['text_1'],
            'pec'          => $r['text_2'],
            'codice_sdi'   => $r['text_3'],
            'indirizzo'    => $r['text_4'],
            'citta'        => $r['text_5'],
            'tipo'         => $r['payload']['tipo'] ?? 'B2B',
            'cf'           => $r['payload']['cf'] ?? null,
            'provincia'    => $r['payload']['provincia'] ?? null,
            'cap'          => $r['payload']['cap'] ?? null,
            'status'       => $r['status'],
            'data_creazione' => $r['date_1'],
            'created_at'   => $r['created_at'],
            'updated_at'   => $r['updated_at'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        $payload = [
            'tipo'      => $b['tipo']      ?? 'B2B',
            'cf'        => $b['cf']        ?? null,
            'provincia' => $b['provincia'] ?? null,
            'cap'       => $b['cap']       ?? null,
        ];
        return [
            'code'    => $b['piva_or_cf']      ?? $b['code']    ?? null,
            'name'    => $b['ragione_sociale'] ?? $b['name']    ?? null,
            'text_1'  => $b['email']           ?? null,
            'text_2'  => $b['pec']             ?? null,
            'text_3'  => $b['codice_sdi']      ?? null,
            'text_4'  => $b['indirizzo']       ?? null,
            'text_5'  => $b['citta']           ?? null,
            'status'  => $b['status']          ?? 'attivo',
            'date_1'  => $b['data_creazione']  ?? date('Y-m-d'),
            'payload' => $payload,
        ];
    }

    /** GET /api/customers/:id/orders */
    public function orders(int $id): array
    {
        $rows = $this->repo->rawAll(
            'SELECT bd.* FROM business_data bd
              JOIN business_relations br ON br.source_id = bd.id AND br.relation_type = ?
              WHERE bd.record_type = ? AND br.target_id = ?
              ORDER BY bd.date_1 DESC',
            ['cliente_di_ordine', 'ordine', $id]
        );
        $orderCtl = new OrderController($this->repo);
        return ['data' => array_map(fn($r) => $orderCtl->dto($r), $rows)];
    }

    /** GET /api/customers/:id/invoices */
    public function invoices(int $id): array
    {
        $rows = $this->repo->rawAll(
            'SELECT bd.* FROM business_data bd
              JOIN business_relations br ON br.source_id = bd.id AND br.relation_type = ?
              WHERE bd.record_type = ? AND br.target_id = ?
              ORDER BY bd.date_1 DESC',
            ['cliente_di_fattura', 'fattura', $id]
        );
        $invCtl = new InvoiceController($this->repo);
        return ['data' => array_map(fn($r) => $invCtl->dto($r), $rows)];
    }
}
