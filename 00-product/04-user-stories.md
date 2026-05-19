# User stories

## Modellrouting

- Som utvecklare vill jag att triviala prompts använder billig modell så att jag sparar tokens.
- Som utvecklare vill jag att komplexa buggar använder stark modell så att risken för fel minskar.
- Som team lead vill jag sätta regler för riskabla filtyper så att känslig kod inte routas fel.
- Som agentbyggare vill jag skicka metadata om tasktyp så att routern inte behöver gissa allt från prompttext.
- Som admin vill jag kunna se varför en modell valdes så att jag kan justera policy.

## Kostnad och budget

- Som admin vill jag sätta maxbudget per tenant, projekt och API key.
- Som utvecklare vill jag se estimerad besparing jämfört med premium-modell på allt.
- Som team lead vill jag få varning när budget närmar sig gräns.

## Fallback

- Som användare vill jag att requesten fortsätter med fallbackmodell om primär provider fallerar.
- Som admin vill jag kunna definiera fallbackkedjor per taskklass.
- Som utvecklare vill jag kunna kräva premium även om billig modell normalt skulle väljas.

## Evals och outcomes

- Som produktägare vill jag jämföra routingstrategier offline innan de aktiveras.
- Som team vill jag skicka feedback om en response accepterades eller avvisades.
- Som plattformsteam vill jag mäta cost per successful task.

## Säkerhet

- Som admin vill jag maska secrets innan prompts skickas till providers.
- Som admin vill jag kunna blockera externa providers för vissa projekt.
- Som organisation vill jag ha audit log över modellval och policyversion.
