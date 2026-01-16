package handlers

import "challenge/kit/broker"

// BusContract defines the publish responsibility used by consumers handlers.
type BusContract = broker.Publisher
