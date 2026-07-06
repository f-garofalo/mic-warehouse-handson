<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class AgenteController extends BaseController
{
    protected function type(): string { return 'agente'; }

    protected function toDto(array $r): array
    {
        return [
            'id'           => $r['id'],
            'codice'       => $r['code'],
            'nome'         => $r['name'],
            'email'        => $r['text_1'],
            'zona'         => $r['text_2'],
            'provvigione'  => $r['amount_1'],
            'status'       => $r['status'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'      => $b['codice'] ?? $b['code'] ?? null,
            'name'      => $b['nome']   ?? $b['name'] ?? null,
            'text_1'    => $b['email']  ?? null,
            'text_2'    => $b['zona']   ?? null,
            'amount_1'  => $b['provvigione'] ?? 0,
            'status'    => $b['status'] ?? 'attivo',
        ];
    }
}
