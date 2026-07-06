<?php
declare(strict_types=1);

namespace MIC\Controllers;

/**
 * BC candidate: Warehouse (extraction target).
 *
 * This controller is in scope for migration to the Go Warehouse BC starting
 * in Phase 06. The PHP implementation here remains the source of truth until
 * that migration completes, after which this file becomes deletable.
 *
 * @see docs/adr/ADR-001-strangler-fig-pattern.md
 * @see phase-02-analysis/ddd-glossary.md (Article aggregate root)
 */
final class ArticleController extends BaseController
{
    protected function type(): string { return 'articolo'; }

    protected function toDto(array $r): array
    {
        return [
            'id'             => $r['id'],
            'sku'            => $r['code'],
            'nome'           => $r['name'],
            'descrizione'    => $r['description'],
            'prezzo_listino' => $r['amount_1'],
            'qta_minima'     => $r['amount_2'],
            'categoria'      => $r['text_1'],
            'iva_default'    => $r['text_2'],
            'unita'          => $r['payload']['unita'] ?? 'pz',
            'peso_kg'        => $r['payload']['peso_kg'] ?? null,
            'status'         => $r['status'],
            'created_at'     => $r['created_at'],
            'updated_at'     => $r['updated_at'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'        => $b['sku'] ?? $b['code'] ?? null,
            'name'        => $b['nome'] ?? $b['name'] ?? null,
            'description' => $b['descrizione'] ?? $b['description'] ?? null,
            'amount_1'    => $b['prezzo_listino'] ?? null,
            'amount_2'    => $b['qta_minima'] ?? 1,
            'text_1'      => $b['categoria'] ?? null,
            'text_2'      => $b['iva_default'] ?? 'IVA22',
            'status'      => $b['status'] ?? 'attivo',
            'payload'     => [
                'unita'   => $b['unita']   ?? 'pz',
                'peso_kg' => $b['peso_kg'] ?? null,
            ],
        ];
    }
}
