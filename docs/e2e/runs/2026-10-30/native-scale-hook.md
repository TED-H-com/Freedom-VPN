# native-scale-hook

Native scale hook smoke: distinct users and peers

Status: pass

Run: 742a2b14-355f-462e-be89-52ffb2c1993b

Source: 05d70d63890bb181937dabfe01b9e9a680d76086

Spec: §8, §19 AT12, SECOPS-10

- counts: {"Users":2,"Peers":4,"IdentityMatches":4,"Authorized":4}
- scope: 2 users x 2 native peers validates the hook; 20 x 3 load is not executed or claimed. The same existing compose/network/image is used.
