# Market Monkey
The hackable liquidity and order flow .... ? (I have no clue what slogan at this point) 

## Start here
Currently the phase where things are moving around. There is not yet a solid structure (especially in the ui) how everything will be organized. That will soon though, once things are starting to look more clear in my brain. 

The way widgets are layed out at this moment in time is just for demo purposes. There will be a dedicated layout system soon.

An important thing to take care of is the that we currently have no internal storing mechanism, hence all data received will be kept in memory. FOR EVER. So if you run the application for 1 day your PC will shutdown. :). Fixing this soon.

## So you want to run the app?
If you want to run the app in the current state, just type the following command in your terminal while being in the root folder:
```
make
```

## What's the plan 
- heatmaps 
- Candles
- Volume profiles
- Footprints
- Indicators
- Custom scription
- Strategies
- ..?

The overall goal would be something like
- ui (package UI so people can extend and build widgets themselves)
- cmd (run server and run the client)
- actor (Hollywood's famous)
- server (We need a server in the future, think about syncing historical data (weekly, monthly, ...))
- client (the app)

## Bugs
Well there will be bugs for sure. You can issue them in the Discord channel. Much appreciated.