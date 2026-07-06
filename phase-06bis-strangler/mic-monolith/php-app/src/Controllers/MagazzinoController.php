<?php
declare(strict_types=1);

namespace MIC\Controllers;

/**
 * BC candidate: Warehouse (extraction target).
 *
 * This controller is in scope for migration to the Go Warehouse BC starting
 * in Phase 07. The PHP implementation here remains the source of truth until
 * that migration completes, after which this file becomes deletable.
 *
 * @see docs/adr/ADR-001-strangler-fig-pattern.md
 * @see phase-02-analysis/ddd-glossary.md (WarehouseLocation value object)
 */
final class MagazzinoController extends BaseController
{
    protected function type(): string { return 'magazzino'; }

    protected function toDto(array $r): array
    {
        return [
            'id'         => $r['id'],
            'codice'     => $r['code'],
            'nome'       => $r['name'],
            'descrizione'=> $r['description'],
            'citta'      => $r['text_1'],
            'cap'        => $r['text_2'],
            'status'     => $r['status'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'        => $b['codice'] ?? $b['code'] ?? null,
            'name'        => $b['nome']   ?? $b['name'] ?? null,
            'description' => $b['descrizione'] ?? null,
            'text_1'      => $b['citta'] ?? null,
            'text_2'      => $b['cap']   ?? null,
            'status'      => $b['status'] ?? 'attivo',
        ];
    }

    /** GET /api/magazzini/:id/giacenze */
    public function giacenze(int $id): array
    {
        // sum movimenti per articolo per magazzino
        $rows = $this->repo->rawAll(
            'SELECT a.id AS articolo_id, a.code AS sku, a.name AS articolo_nome,
                    COALESCE(SUM(br_art.amount), 0) AS giacenza
               FROM business_data m
               JOIN business_relations br_mag ON br_mag.target_id = m.id AND br_mag.relation_type = ?
               JOIN business_data mov ON mov.id = br_mag.source_id AND mov.record_type = ?
               JOIN business_relations br_art ON br_art.source_id = mov.id AND br_art.relation_type = ?
               JOIN business_data a ON a.id = br_art.target_id AND a.record_type = ?
              WHERE m.id = ?
              GROUP BY a.id, a.code, a.name
              ORDER BY giacenza DESC
              LIMIT 200',
            ['movimento_in_magazzino', 'movimento', 'movimento_di_articolo', 'articolo', $id]
        );
        return ['data' => array_map(fn($r) => [
            'articolo_id'   => (int)$r['articolo_id'],
            'sku'           => $r['sku'],
            'articolo_nome' => $r['articolo_nome'],
            'giacenza'      => (float)$r['giacenza'],
        ], $rows)];
    }
}
