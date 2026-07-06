<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class CategoryController extends BaseController
{
    protected function type(): string { return 'categoria'; }

    protected function toDto(array $r): array
    {
        return [
            'id'         => $r['id'],
            'codice'     => $r['code'],
            'nome'       => $r['name'],
            'descrizione'=> $r['description'],
            'status'     => $r['status'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'        => $b['codice'] ?? $b['code'] ?? null,
            'name'        => $b['nome']   ?? $b['name'] ?? null,
            'description' => $b['descrizione'] ?? $b['description'] ?? null,
            'status'      => $b['status'] ?? 'attivo',
        ];
    }
}
