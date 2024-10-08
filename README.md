# Reservation rest api

## Scenario

We have two kinds of users, providers and clients. Providers have a schedule where they are available to see clients. Clients want to book an appointment time, in advance, from that schedule.

## Task

Build an API (e.g. RESTful) with the following endpoints:

- Allows providers to submit times they are available for appointments
- - e.g. On Friday the 13th of August, Dr. Jekyll wants to work between 8am and 3pm
- Allows a client to retrieve a list of available appointment slots
- - Appointment slots are 15 minutes long
- Allows clients to reserve an available appointment slot
- Allows clients to confirm their reservation

Additional Requirements:

- Reservations expire after 30 minutes if not confirmed and are again available for other clients to reserve that appointment slot
- Reservations must be made at least 24 hours in advance

# Tools

- golang
- docker
- [gin](https://github.com/gin-gonic/gin): go based http server
- [openapi](https://www.openapis.org/): schema for rest api
- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen): generates client and server code from open-api file
- [postgres](https://www.postgresql.org): provider and client data
- [nix](https://nixos.org/): local development environment and ci environment consistency
- [go-example](https://github.com/tateexon/go-example): my golang project template that comes with the nix environment, linting, githooks, and basic ci

# How to run locally

```sh
# Build reservation container, then start the db, the reservation api and the swagger ui via docker compose
make run-local
```

You can then access the api through "http://localhost:8080" or you can test out the api via the swagger ui through "http://localhost:8081".

# Open API documentation for api

The openapi schema can be found [here](./schema/openapi.yaml). The api can also be viewed through the swagger ui in the method provided in the "How to run locally" section abvoe. For a quick reference of the calls:

## Creating clients and providers

- POST /users Create a new client or provider
- GET /users/{userId} Get user details

## Provider

- POST /providers/{providerId}/availability Submit provider availability, will round up to closest 15 minute interval as a start time and down on the end time

## Appointments

- GET /appointments Get available appointment slots
- POST /appointments Reserve an appointment slot
- POST /appointments/{appointmentId}/confirm Confirms a reservation

# Notes:

This is pretty much a brute force implementation due to the time constraints. I like to write code in three steps: make it work, make it right, make it fast. I never fully made it out of the make it work phase.

I spent probably half of the time researching the tools and their changes since I last used some of them. And then another good chunk getting the environment setup so I could iterate quickly, namely, docker compose with swagger so I could manually try stuff, oapi-codegen so I could regenerate my api, dbeaver downloaded and hooked up to postgres so I could build my db with a gui. The other key part was getting testcontainers-go setup in tests so I could quickly spin up the db and run my code against it. With that all in place I probably only had about an hour left to tackle the logic and write this readme.

## assumptions

- I didn't put a lot of thought into creating clients and providers. In a real world scenario I would expect us to have the signup and such already in place so we would already have clients and providers in the database to use.
- I didn't handle security stuff like whether a specific client/provider should be able to make a call or not. That should be able to be handled and added fairly easily based on what is already existing in your system.

## the good

- The environment is setup to be able to quickly iterate and debug and because of nix there should be no "it works on my box" assuming you use nix to develop with.
- The basic functionality from the requirements is working so long as you stay on happy paths.
- The project is using openapi as a standard. There are many tools openapi has available to build and test rest services. There are many other tools that use it as well including tools like the zap security testing tool.

## the bad

- I didn't have enough time to clean up my code so it is not super well organized.
- A pre-commit check to make sure the generated code is up to date with the openapi.yaml would be very nice to have but we don't have one yet. Basically it would keep us from shooting ourselves in the foot.
- I didn't have enough time to harden the code. Hindsight tells me I should have written much more defensive code up front since I wasn't going to have time to harden it. Because of this I am sure there are lots of little bugs, especially around time zones, that were missed.
- While I have tests I don't have the coverage I really should have. There is also a lot of copy paste because I was just trying to fly through it at the end. Copy paste errors very likely exist causing some parts of tests to maybe not test exactly what I think they are testing.
- The database setup and my queries are not optimized. If this underwent any kind of performance testing I am pretty sure we would find some of my queries take far longer then they should.

## other stuff before production ready

- Add proper tests, a decent portion are there but there should be a proper walk through to cover more edge cases, especially around times zones.
- Do some hardening to see if any db queries are taking longer then they should and optimize those.
- Probably redesign how to handle reservation expiration after 30 minutes. I simplified a bit in this project by doing the math in the queries but this could probably be better handled with a cron job.
- Add more ci around the code generation.
- Add a proper way to configure the api, right now it takes environment variables "POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, and POSTGRES_URL" but this is not a real secure way to go about it and it should probably pull these from a vault or secrets manager.
