<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class PagamentoController extends BaseController
{
    protected function type(): string { return 'pagamento'; }

    protected function toDto(array $r): array
    {
        $fat = null;
        if (!empty($r['id'])) {
            $fat = $this->repo->rawOne(
                'SELECT bd.id, bd.code FROM business_relations br
                  JOIN business_data bd ON bd.id = br.target_id
                 WHERE br.source_id = ? AND br.relation_type = ? LIMIT 1',
                [(int)$r['id'], 'pagamento_di_fattura']
            );
        }
        return [
            'id'            => $r['id'],
            'codice'        => $r['code'],
            'descrizione'   => $r['name'],
            'data'          => $r['date_1'],
            'importo'       => $r['amount_1'],
            'metodo'        => $r['text_1'],
            'fattura_id'    => $fat['id'] ?? null,
            'fattura_numero'=> $fat['code'] ?? null,
            'status'        => $r['status'],
        ];
    }

    protected function fromPayload(array $b): array
    {
        return [
            'code'     => $b['codice'] ?? $b['code'] ?? ('PAG-'.uniqid()),
            'name'     => $b['descrizione'] ?? 'Pagamento',
            'date_1'   => $b['data'] ?? date('Y-m-d'),
            'amount_1' => $b['importo'] ?? 0,
            'text_1'   => $b['metodo'] ?? 'Bonifico',
            'status'   => $b['status'] ?? 'confermato',
        ];
    }

    public function create(): array
    {
        $body = $this->readJsonBody();
        if (!$body) return [400, ['error' => 'invalid_body']];
        $fattura_id = (int)($body['fattura_id'] ?? 0);
        if (!$fattura_id) return [400, ['error' => 'fattura_id richiesto']];
        $b = $this->repo->create('pagamento', $this->fromPayload($body));
        $this->repo->relate($b->id, $fattura_id, 'pagamento_di_fattura', (float)($body['importo'] ?? 0));
        return [201, ['data' => $this->toDto($b->toArray())]];
    }
}
