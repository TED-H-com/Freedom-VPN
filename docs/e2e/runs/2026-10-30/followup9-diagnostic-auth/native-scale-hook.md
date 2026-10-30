# native-scale-hook

Native scale hook smoke: distinct users and peers

Status: pass

Run: 376e5f84-535e-4ddd-902b-985852d4691b

Source: 724ab19fd46a974a66d138ac0cddcd7f803bafb7

Spec: §8, §19 AT12, SECOPS-10

- counts: {"Users":2,"Peers":4,"IdentityMatches":4,"Authorized":4}
- scope: 2 users x 2 native peers validates the hook; 20 x 3 load is not executed or claimed. The same existing compose/network/image is used.
