package board

// standardPorts places nine ports around the coast, with one or two unused
// coastal paths between them. There are four generic ports and one per resource.
func standardPorts() map[PathCoord]ResourceType {
	return map[PathCoord]ResourceType{
		NewPathCoord(CrossCoord{2, 0}, CrossCoord{3, 0}):  ResourceInvalid,
		NewPathCoord(CrossCoord{4, 1}, CrossCoord{4, 2}):  ResourceWheat,
		NewPathCoord(CrossCoord{5, 3}, CrossCoord{5, 4}):  ResourceOre,
		NewPathCoord(CrossCoord{5, 7}, CrossCoord{5, 8}):  ResourceInvalid,
		NewPathCoord(CrossCoord{3, 9}, CrossCoord{4, 9}):  ResourceSheep,
		NewPathCoord(CrossCoord{2, 9}, CrossCoord{2, 10}): ResourceInvalid,
		NewPathCoord(CrossCoord{0, 7}, CrossCoord{0, 8}):  ResourceInvalid,
		NewPathCoord(CrossCoord{0, 4}, CrossCoord{0, 5}):  ResourceBrick,
		NewPathCoord(CrossCoord{0, 2}, CrossCoord{1, 2}):  ResourceWood,
	}
}
