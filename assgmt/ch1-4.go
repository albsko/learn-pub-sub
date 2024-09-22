package main

func (u user) doBattles(subCh <-chan move) []piece {
	piecesInBattle := make([]piece, 0, 1024)
	for mv := range subCh {
		for _, piece := range u.pieces {
			if piece.location == mv.piece.location {
				piecesInBattle = append(piecesInBattle, piece)
			}
		}
	}

	return piecesInBattle
}

// don't touch below this line

type user struct {
	name   string
	pieces []piece
}

type move struct {
	userName string
	piece    piece
}

type piece struct {
	location string
	name     string
}

func (u user) march(p piece, publishCh chan<- move) {
	publishCh <- move{
		userName: u.name,
		piece:    p,
	}
}

func distributeBattles(publishCh <-chan move, subChans []chan move) {
	for mv := range publishCh {
		for _, subCh := range subChans {
			subCh <- mv
		}
	}
}
