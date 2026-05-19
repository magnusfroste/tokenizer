# Produktvision

## Vision

Utvecklare, agenter och applikationer ska inte behöva välja modell manuellt för varje prompt. En snabb router ska automatiskt välja lämplig modell utifrån uppgiftens krav, risk och kontext.

## Produktformulering

**Model-router är en låg-latency gateway som routar varje prompt till rätt modell med policy, kostnadskontroll, fallback och outcome-feedback.**

## Primära användare

- Solo-utvecklare som vill sänka tokenkostnader utan att tappa kvalitet.
- Små team som använder flera AI-kodverktyg och behöver central kostnadskontroll.
- Plattformsteam som vill styra vilka modeller som får användas för olika typer av data och uppgifter.
- Agentbyggare som behöver dynamiskt modellval i agent-loopen.

## Kärnproblem

Manuellt modellval är långsamt och opålitligt:

- Dyra modeller används på triviala uppgifter.
- Billiga modeller används ibland på riskabla uppgifter.
- Modellbyte kräver mental overhead och konfiguration.
- Team saknar spårbarhet kring varför en modell användes.
- Det är svårt att veta vilka modeller som faktiskt lyckas på en viss tasktyp.

## Produktlöfte

Routern ska göra tre saker bättre än användaren gör manuellt:

1. Välja rimlig modell snabbare.
2. Reducera kostnad utan att öka teknisk risk.
3. Lära sig från faktiska outcomes.

## Icke-förhandlingsbara principer

- Routinglagret får inte kännas långsamt.
- För hög-risk-uppgifter ska routern vara konservativ.
- Alla beslut ska vara förklarbara i loggar.
- Policy ska kunna överstyra scoring.
- Providerfel ska inte stoppa arbetet om fallback finns.
- Kunddata ska inte tränas på utan explicit opt-in.

## Första produktnisch

Fokusera initialt på **AI-kodagenter och utvecklararbetsflöden**. Det är en bättre nisch än generell chatt-routing eftersom outcomes ofta kan mätas objektivt via tester, typecheck, diff-acceptance och användarfeedback.
