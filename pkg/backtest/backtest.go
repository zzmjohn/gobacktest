// Package backtest provides a simple stock backtesting framework.
package backtest

import (
	"log"
)

// Test is a basic back test struct
type Test struct {
	symbols    []string
	data       DataHandler
	strategy   StrategyHandler
	portfolio  PortfolioHandler
	exchange   ExecutionHandler
	statistic  StatisticHandler
	eventQueue []EventHandler
}

// New creates a default test backtest value for use.
func New() *Test {
	return &Test{}
}

// SetSymbols sets the symbols to include into the test
func (t *Test) SetSymbols(symbols []string) {
	t.symbols = symbols
}

// SetData sets the data provider to to be used within the test
func (t *Test) SetData(data DataHandler) {
	t.data = data
}

// SetStrategy sets the strategy provider to to be used within the test
func (t *Test) SetStrategy(strategy StrategyHandler) {
	t.strategy = strategy
}

// SetPortfolio sets the portfolio provider to to be used within the test
func (t *Test) SetPortfolio(portfolio PortfolioHandler) {
	t.portfolio = portfolio
}

// SetExchange sets the execution provider to to be used within the test
func (t *Test) SetExchange(exchange ExecutionHandler) {
	t.exchange = exchange
}

// SetStatistic sets the statistic provider to to be used within the test
func (t *Test) SetStatistic(statistic StatisticHandler) {
	t.statistic = statistic
}

// Run starts the test.
func (t *Test) Run() error {
	log.Println("Running backtest:")
	log.Printf("Counting %v data events. \n", len(t.data.Stream()))

	// before first run, set portfolio cash
	t.portfolio.SetCash(t.portfolio.InitialCash())

	// poll event queue
	for event, ok := t.nextEvent(); true; event, ok = t.nextEvent() {
		// no event in queue
		if !ok {
			// poll data stream
			data, ok := t.data.Next()
			// no  data event, exit event loop
			if !ok {
				break
			}
			// found data, add to event stream
			t.eventQueue = append(t.eventQueue, data)
			// start new event polling cycle
			continue
		}

		// processing event
		err := t.eventLoop(event)
		if err != nil {
			return err
		}
		// event in queue found, add to event history
		t.statistic.TrackEvent(event)
	}

	t.statistic.PrintResult()

	return nil
}

// nextEvent gets the next event from the events queue
func (t *Test) nextEvent() (event EventHandler, ok bool) {
	// if event queue empty return false
	if len(t.eventQueue) == 0 {
		return event, false
	}

	// return first element from the event queue
	event = t.eventQueue[0]
	t.eventQueue = t.eventQueue[1:]

	return event, true
}

// eventLoop
func (t *Test) eventLoop(e EventHandler) error {
	// type check for event type
	switch event := e.(type) {
	case DataEventHandler:
		// update portfolio to the last known price data
		t.portfolio.Update(event)
		// update statistics
		t.statistic.Update(event, t.portfolio)

		signal, err := t.strategy.CalculateSignal(event, t.data, t.portfolio)
		if err != nil {
			break
		}
		t.eventQueue = append(t.eventQueue, signal)

	case SignalEvent:
		order, err := t.portfolio.OnSignal(event, t.data)
		if err != nil {
			break
		}
		t.eventQueue = append(t.eventQueue, order)

	case OrderEvent:
		fill, err := t.exchange.ExecuteOrder(event, t.data)
		if err != nil {
			break
		}
		t.eventQueue = append(t.eventQueue, fill)
	case FillEvent:
		transaction, err := t.portfolio.OnFill(event, t.data)
		if err != nil {
			break
		}
		t.statistic.TrackTransaction(transaction)
	}

	return nil
}