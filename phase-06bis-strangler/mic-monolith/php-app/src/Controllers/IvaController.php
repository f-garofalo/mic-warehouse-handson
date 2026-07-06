<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class IvaController extends BaseController
{
    protected function type(): string { return 'aliquota_iva'; }

    protected function toDto(array $r): array
    {
        return [
            'id'           => $r['id'],
            'codice'       => $r['code'],
            'nome'         => $r['name'],
            'descrizione'  => $r['description'],
            'percentuale'  => $r['amount_1'],
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
            'status'      => $b['status'] ?? 'attivo',
        ];
    }
}
