# native-scale-hook

Native scale hook smoke: distinct users and peers

Status: pass

Run: 65712989-0b05-4117-9a50-689499cd31be

Source: 43176b8860bf90970e96b100c1ba4e957f09363d

Spec: §8, §19 AT12, SECOPS-10

- counts: {"Users":2,"Peers":4,"IdentityMatches":4,"Authorized":4}
- scope: 2 users x 2 native peers validates the hook; 20 x 3 load is not executed or claimed. The same existing compose/network/image is used.
