# native-scale-hook

Native scale hook smoke: distinct users and peers

Status: pass

Run: dc2f6353-89ec-434c-bc68-567066d9fe51

Source: b15ff057cd5e91e227477de6a09d977931fa62d8

Spec: §8, §19 AT12, SECOPS-10

- counts: {"Users":2,"Peers":4,"IdentityMatches":4,"Authorized":4}
- scope: 2 users x 2 native peers validates the hook; 20 x 3 load is not executed or claimed. The same existing compose/network/image is used.
