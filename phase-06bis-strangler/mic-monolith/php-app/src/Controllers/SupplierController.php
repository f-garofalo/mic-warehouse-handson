<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class SupplierController extends BaseController
{
    protected function type(): string { return 'fornitore'; }

    protected function toDto(array $r): array
    {
        return [
            'id'              => $r['id'],
            'piva'            => $r['code'],
            'ragione_sociale' => $r['name'],
            'email'           => $r['text_1'],
            'pec'             => $r['text_2'],
            'indirizzo'       => $r['text_4'],
            'citta'           => $r['text_5'],
            'provincia'       => $r['payload']['provincia'] ?? null,
            'cap'             => $r['payload']['cap'] ?? null,
            'status'          => $r['status'],
            'created_at'      => $r['created_at'],
            'updated_at'      => $r['updated_at'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'    => $b['piva'] ?? $b['code'] ?? null,
            'name'    => $b['ragione_sociale'] ?? $b['name'] ?? null,
            'text_1'  => $b['email'] ?? null,
            'text_2'  => $b['pec']   ?? null,
            'text_4'  => $b['indirizzo'] ?? null,
            'text_5'  => $b['citta'] ?? null,
            'status'  => $b['status'] ?? 'attivo',
            'payload' => [
                'provincia' => $b['provincia'] ?? null,
                'cap'       => $b['cap']       ?? null,
            ],
        ];
    }
}
