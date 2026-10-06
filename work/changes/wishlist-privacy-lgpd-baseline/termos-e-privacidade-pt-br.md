# Wishlist — minuta de Termos de Uso e Aviso de Privacidade

**Versão para revisão jurídica — 6 de outubro de 2026. Não publicar como texto aprovado.**

Esta minuta descreve o produto e as decisões registradas no repositório nesta data. Algumas bases legais, condições de publicação de sugestões, informações sobre fornecedores e prazos de conservação fora do banco de dados ainda dependem de validação. O texto deve ser ajustado à operação efetivamente implantada antes de ser apresentado aos visitantes.

## 1. Termos de Uso — proposta

### 1.1. Quem oferece o Wishlist

O Wishlist é oferecido por **Henrique Riccio Desenvolvimento de Software LTDA**, CNPJ **64.475.365/0001-55**, nome de divulgação **Wastingnotime**, com sede em São Paulo/SP. O canal para dúvidas sobre dados pessoais é [privacy@wastingnotime.org](mailto:privacy@wastingnotime.org).

### 1.2. Finalidade do serviço

O Wishlist apresenta ideias de funcionalidades de produtos Wastingnotime e permite que visitantes verifiquem um endereço de e-mail para votar ou enviar sugestões. Os votos são um sinal público de interesse. A classificação e a quantidade de votos ajudam a avaliação humana da Wastingnotime, mas **não constituem promessa de desenvolvimento, prazo de entrega ou inclusão no roteiro de produto**. Somente a equipe autorizada decide se uma ideia será produzida, entregue, alterada ou retirada.

### 1.3. Acesso e participação

A consulta ao quadro público dispensa identificação. Para votar ou enviar uma sugestão, o visitante informa um endereço de e-mail e confirma seu controle por um código de uso único. A verificação permite associar votos e sugestões a uma identidade privada e impedir votos duplicados da mesma identidade em uma funcionalidade. O endereço de e-mail e a identidade de quem votou não aparecem nas respostas públicas do serviço; o público vê contagens agregadas.

O visitante deve usar um endereço de e-mail sob seu controle e enviar sugestões pertinentes ao produto. O campo livre não deve incluir dados pessoais desnecessários, dados sensíveis, dados de terceiros ou conteúdo que o visitante não possa legitimamente compartilhar. A equipe pode revisar, editar, combinar, rejeitar ou deixar de publicar sugestões. **[Revisão jurídica: definir regras objetivas de conteúdo, procedimento de moderação e eventuais medidas contra abuso.]**

### 1.4. Sugestões e conteúdo público

Uma sugestão enviada é inicialmente privada e acessível aos revisores autorizados. Se for aceita, seu título e sua descrição, após revisão, podem originar uma funcionalidade visível no quadro público. A equipe deve retirar dados pessoais e detalhes de terceiros antes da publicação. Também pode associar a sugestão a uma funcionalidade existente sem publicar o texto privado. Uma funcionalidade já publicada pode permanecer visível após a eliminação do registro privado de origem; nesse caso, seu texto deve ser examinado separadamente se houver pedido relacionado a dados pessoais.

**[Revisão jurídica: definir autorização de uso do conteúdo enviado, direitos de propriedade intelectual, atribuição ou ausência de atribuição, e se a publicação exige uma escolha específica do visitante.]** Esta minuta não presume transferência de direitos autorais nem consentimento para outros fins.

### 1.5. Disponibilidade e alterações

O quadro, as contagens e os estados das funcionalidades podem mudar. Um voto pode ser retirado pelo visitante ou deixar de ser contado quando expirar o período de conservação vinculado à última verificação do e-mail. Essa mudança não altera automaticamente o estado de uma funcionalidade. **[Revisão jurídica: completar regras de disponibilidade, alterações dos termos e lei/foro aplicáveis, se cabíveis.]**

## 2. Aviso de Privacidade — proposta

### 2.1. Controladora e contato

A controladora proposta para os dados dos visitantes do Wishlist é **Henrique Riccio Desenvolvimento de Software LTDA**, CNPJ **64.475.365/0001-55**, nome de divulgação Wastingnotime, em São Paulo/SP. Pedidos e dúvidas sobre privacidade podem ser enviados a [privacy@wastingnotime.org](mailto:privacy@wastingnotime.org). Henrique Riccio atende esse canal em nome da empresa. A indicação desse responsável pelo atendimento **não equivale, por si só, à designação formal de encarregado**.

### 2.2. Dados e finalidades

| Atividade | Dados tratados | Finalidade | Hipótese legal proposta para validação |
| --- | --- | --- | --- |
| Verificação do e-mail | Endereço, código de uso único protegido no banco, tentativas e horários de solicitação/verificação | Permitir voto e sugestão mediante prova de controle do endereço; limitar abuso | Legítimo interesse, art. 7º, IX, da LGPD, sujeito a avaliação de necessidade e balanceamento; avaliar se outra hipótese descreve melhor o serviço solicitado. |
| Sessão de acesso | Identificador privado, endereço e token de sessão | Manter o acesso verificado por prazo limitado | Legítimo interesse, art. 7º, IX, sujeito a avaliação. |
| Voto | Relação privada entre identidade e funcionalidade, com data; contagem pública agregada | Evitar voto duplicado e medir interesse por ideias | Legítimo interesse, art. 7º, IX, sujeito a avaliação. |
| Sugestão privada | Identidade, produto, título, descrição, estado e datas de revisão | Receber e avaliar ideias | Legítimo interesse, art. 7º, IX, sujeito a avaliação. |
| Publicação de ideia revisada | Título, descrição e metadados da funcionalidade | Mostrar ao público uma ideia selecionada | **Hipótese legal ainda não definida**; avaliar legítimo interesse com salvaguardas e transparência ou escolha específica do visitante. |
| Segurança e operação | Possíveis dados de conexão, requisição, erro e entrega de e-mail; campos e destinos ainda não inventariados | Diagnosticar falhas, prevenir abuso e investigar incidentes | Legítimo interesse proporcional, sujeito a avaliação; obrigação legal apenas quando identificada uma obrigação concreta. |
| Atendimento de direitos | Contato, prova de verificação, pedido, decisão e providências | Identificar o solicitante e responder ao pedido | Cumprimento de obrigação legal, art. 7º, II, sujeito à confirmação da obrigação e do registro mínimo necessário. |

O código enviado ao e-mail confirma o controle do endereço; **não é consentimento para marketing**. O inventário atual não prevê publicidade comportamental, análise de uso opcional ou envio de mensagens promocionais. Qualquer finalidade nova exige avaliação e informação próprias.

### 2.3. Quem acessa e onde os dados circulam

O navegador se comunica com uma camada intermediária do próprio site, que encaminha as operações à API do Wishlist. A API guarda registros de visitantes em PostgreSQL. O provedor de e-mail **Mailgun** recebe o endereço e a mensagem necessários para entregar o código. Revisores autorizados da Wastingnotime acessam sugestões privadas para moderação. O público acessa funcionalidades e totais de votos, sem e-mails, sessões ou vínculos de votos com identidades.

**[Confirmar antes da publicação:]** identidade contratual dos operadores de hospedagem, banco, registros técnicos e cópias de segurança; locais de tratamento; termos com fornecedores; existência e mecanismo de transferências internacionais; campos efetivos de logs e acesso administrativo. O Casdoor participa da autenticação dos administradores e requer inventário próprio ou vínculo documentado com aviso organizacional.

### 2.4. Conservação e exclusão

Os prazos a seguir foram escolhidos para o produto e **não são prazos impostos pela LGPD**. A implementação existe no código da API; sua implantação e operação em produção ainda precisam ser confirmadas.

| Registro no banco principal | Regra prevista |
| --- | --- |
| Código de verificação | Válido por 10 minutos, com até cinco tentativas incorretas; excluído após uso ou substituição, ou até 24 horas depois de expirar. |
| Histórico de solicitações do código | Janela de controle de uma hora; exclusão de registros com mais de 24 horas. |
| Sessão | Válida por 30 dias; token expirado excluído em até 24 horas. |
| E-mail e identidade sem voto ou sugestão conservada | Exclusão 90 dias após o fim da validade da última sessão, se não ocorrer nova verificação. |
| Voto vinculado à identidade | Exclusão 12 meses após a última verificação bem-sucedida do e-mail, se não houver nova verificação. O total público pode diminuir. |
| Sugestão privada ainda não analisada | Meta de análise em 90 dias; exclusão após 180 dias do envio se permanecer pendente. |
| Sugestão privada analisada | Exclusão 12 meses após a análise, ressalvada exceção específica e documentada. |
| Texto público de funcionalidade | Revisão para retirar dados pessoais antes de publicar; conservação enquanto a funcionalidade estiver publicada, com revisão anual proposta. |

A meta para logs rotineiros da aplicação e do proxy é de **até 30 dias**, mas sua configuração e os destinos reais ainda não foram verificados. O provedor de e-mail pode conservar eventos de entrega, corpos de mensagens e registros de supressão segundo regras distintas que ainda devem ser apuradas. A infraestrutura possui uma política aplicada para cópias diárias do PostgreSQL com expiração em **28 dias**, mas a execução agendada, os alertas, as demais cópias e a reconciliação de exclusões após restauração ainda exigem verificação. Essas condições impedem uma promessa geral de eliminação de todas as cópias no momento da exclusão do banco principal.

Uma exceção de conservação deverá identificar fundamento, finalidade, dados afetados, responsável, acesso restrito e data de término. **[Revisão jurídica e operacional: aprovar prazos, exceções, logs, fornecedor de e-mail, cópias de segurança e procedimento após restauração.]**

### 2.5. Direitos do titular

Na área `/privacy`, após uma **nova verificação por código**, o visitante pode consultar o e-mail, as datas da conta, as sessões ativas, os votos e as sugestões privadas associados a sua identidade; corrigir o e-mail mediante verificação do novo endereço; ou solicitar a exclusão dos registros vinculados no banco principal. A verificação recente é exigida para proteger essas operações. A correção encerra as outras sessões; a exclusão encerra todas as sessões e reduz os totais de votos correspondentes.

Também é possível escrever para [privacy@wastingnotime.org](mailto:privacy@wastingnotime.org), inclusive para pedir revisão de texto público que possa identificar alguém ou tratar dados fora do banco principal. A equipe verificará a identidade do solicitante de forma proporcional antes de revelar ou alterar dados. O atendimento humano e eventuais exceções serão documentados em registro privado. Outros direitos previstos na LGPD podem ser exercidos pelo mesmo canal e serão analisados conforme o caso.

A exclusão automatizada retira do banco principal a identidade, o e-mail, as sessões, os códigos e seu histórico, os votos e as sugestões privadas vinculadas. Um texto público derivado de sugestão é encaminhado para revisão humana; registros de provedores, logs, arquivos legados e cópias de segurança requerem tratamento próprio. A resposta ao titular deve indicar o que foi efetivamente concluído e o que ainda depende dessas etapas.

### 2.6. Crianças, adolescentes e dados sensíveis

O Wishlist não foi concebido para coletar dados sensíveis nem para solicitar dados de crianças ou adolescentes. Entretanto, o campo livre pode receber essas informações. O produto deve orientar o visitante a não incluí-las e a equipe deve revisar, retirar ou restringir conteúdo indevido. **[Revisão jurídica: definir público pretendido, tratamento de menores e procedimento quando tais dados forem recebidos.]**

## 3. Pontos para decisão do advogado e da equipe operacional

1. Validar a identificação da controladora e o registro oficial, além da redação dos Termos e do Aviso.
2. Escolher e documentar as hipóteses legais por finalidade, especialmente o teste de legítimo interesse e a publicação de sugestões; definir a autorização de uso do conteúdo enviado.
3. Avaliar a incidência territorial do GDPR antes do acesso público sem restrição geográfica. A prioridade inicial da LGPD não resolve essa questão.
4. Confirmar se a empresa pode utilizar a dispensa de designação formal de encarregado para agente de pequeno porte e se o atendimento pelo canal indicado satisfaz as obrigações aplicáveis. Reavaliar a classificação de risco quando escala, público ou tratamento mudarem.
5. Aprovar os prazos de conservação, as exceções, o registro privado de pedidos, a revisão de conteúdo público e a reconciliação de exclusões após restauração de backup.
6. Confirmar fornecedores, destinos e campos de logs, localização dos dados, transferências internacionais, retenção no Mailgun e funcionamento do canal de privacidade.
7. Conferir a versão efetivamente implantada antes de converter esta minuta em documento público e em aviso breve junto ao campo de e-mail.

## 4. Fontes para a revisão

- [Proposta de tratamento e bases legais](processing-and-legal-basis.md), [proposta de conservação e exclusão](retention-and-deletion.md), [fluxo de direitos do titular](rights-requests.md) e [prontidão de produção](../wishlist-production-readiness/production_readiness.md).
- [Lei nº 13.709/2018, texto compilado](https://www.planalto.gov.br/ccivil_03/_Ato2015-2018/2018/Lei/L13709compilado.htm), especialmente arts. 6º, 7º, 9º, 10, 15 a 18 e 41.
- [Guia da ANPD sobre legítimo interesse](https://www.gov.br/anpd/pt-br/centrais-de-conteudo/materiais-educativos-e-publicacoes/copy_of_guia_legitimo_interesse.pdf) e [Resolução CD/ANPD nº 2/2022](https://www.gov.br/anpd/pt-br/acesso-a-informacao/institucional/atos-normativos/regulamentacoes_anpd/resolucao-cd-anpd-no-2-de-27-de-janeiro-de-2022).
