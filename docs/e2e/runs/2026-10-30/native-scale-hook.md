# native-scale-hook

Native scale hook smoke: distinct users and peers

Status: pass

Run: 22c14cd9-39fc-444b-a7c5-c04cf3ffbff6

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

Spec: §8, §19 AT12, SECOPS-10

- counts: {"Users":2,"Peers":4,"IdentityMatches":4,"Authorized":4}
- scope: 2 users x 2 native peers validates the hook; 20 x 3 load is not executed or claimed. The same existing compose/network/image is used.
