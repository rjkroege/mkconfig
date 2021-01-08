# TL;DR
There is a command called `mkconfig`. `mk` runs it to receive a bunch
of variable definitions that it will use.

# Longer Overview
I needed some way to customize how `mk` would copy and install
state. I eventually decided that the easiest way to do this would be
to have a Golang program that would know how to do this.

Most of the defined variables are reasonably self-explanatory. The
`mkconfig` program makes decisions based on what else is installed
on the system and the `GOOS` and `GOARCH` variables to figure out
which binaries from what path need to be installed to what target.

However, I decided that I wanted to not keep my selection of binaries
in unsecured storage. At least not storage visible to the entire
public Internet. So I decided to require authentication to download
the binaries from GCS (`mk` does that) and meant that `mkconfig`
would need to provide an appropriate access token to `mk`

Getting access tokens needs the OAuth dance. See below.

I let some featurism in by adding the actual downloading code as well
because it's so little additional code after actually doing the OAuth
dance. Also it tests the OAuth token and means that I don't have to
worry about escaping the URLs or accesstoken contents.

# OAuth Dance
I started writing this from some convenient (though probably wrong)
example on stackoverflow. It was not sufficient. I needed to be more
principled. I consulted [Martin Fowler's
article](https://martinfowler.com/articles/command-line-google.html)
for learning. It had the benefit of explaining things reasonably
clearly.

A command line app is most like [Google's mobile app
flow](https://developers.google.com/identity/protocols/oauth2/native-app).
I think. This is the key take-away.

It uses the "cut&paste" approach that Google claims is deprecated.
Conceivably, I could use IOT/TV scheme instead except it says that
this supports a more limited set of scopes. I have implemented the
"cut&paste" flow.

## Ideas

* what are the best practices for handling the *OAuth Client ID*? How can the app
be distributed without having the client-id compiled into it?

* Internet is imprecise about this. A distributed app would have the
client-id compiled into it?

* the client-id identifies the app. A malicious developer could rip the
client-id out of the app and use that in a different app. Then, when a 
user uses the new malicious app, it could masquerade as the original
app and phish the user.

* presumably Google's inconvenient app verification process exists
for precisely this reason. But: what are the rules and such that the
Google imposes to limit this eventuality?

* I will store the client-id separately in a text file.

* This text file can be downloaded again from the GCE app console.

* The OAuth dance uses the client-id to obtain *authorization* and *refresh*
tokens. 

* The auth token and refresh token should be kept somewhere
secure. Like keychain on Mac. Or the equivalent on Linux. Because
the refresh token permits an unlimited number of access tokens to
be vended and the auth token gives "logged-in" access to the scopes
specified in the OAuth dance.

* So, where do I keep the client secret? I  used a
[go keychain library](https://github.com/keybase/go-keychain) to
store these secrets on MacOS.

* There isn't a version of keychain on Linux. Instead, I encrypt the
configuration locally in a file.

* I don't need to keep the original OAuth config? Yes. I can download it
again.

# Design Discussion

* should this tool (e.g. `mkconfig`) also implement the downloading?

* is trivial to add and would put more of the code in Go (instead of
shell) where it's easier to make sure that it actually works exactly
like it's suppose to?

* yes. Because I like this idea and am worrying about how `mk` and
shell will handle the quoting of rules and tokens.


