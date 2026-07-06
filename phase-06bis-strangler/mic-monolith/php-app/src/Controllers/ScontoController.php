<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class ScontoController extends BaseController
{
    protected function type(): string { return 'sconto'; }

    protected function toDto(array $r): array
    {
        return [
            'id'           => $r['id'],
            'codice'       => $r['code'],
            'nome'         => $r['name'],
            'descrizione'  => $r['description'],
            'percentuale'  => $r['amount_1'],
            'tipo'         => $r['text_1'],
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
            'amount_1'    => $b['percentuale'] ?? 0,
            'text_1'      => $b['tipo'] ?? 'percentuale',
            'date_1'      => $b['valido_da'] ?? null,
            'date_2'      => $b['valido_a']  ?? null,
            'status'      => $b['status'] ?? 'attivo',
        ];
    }
}
