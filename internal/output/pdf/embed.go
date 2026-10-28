package pdf

import _ "embed"

//go:embed static/award.jpeg
var awardJPEG []byte

//go:embed static/fonts/LiberationSans-Regular.ttf
var fontRegular []byte

//go:embed static/fonts/LiberationSans-Bold.ttf
var fontBold []byte

func AwardJPEG() []byte { return awardJPEG }
