package stubhandlers

import "testing"

func TestValidateAttributionCode(t *testing.T) {
	validCodes := []struct {
		In  string
		Out string
	}{
		{
			"source%3Dgoogle.com%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"campaign=%28not+set%29&content=%28not+set%29&medium=organic&source=google.com",
		},
	}
	for _, c := range validCodes {
		res, err := validateAttributionCode(c.In)
		if err != nil {
			t.Errorf("err: %v, code: %s", err, c.In)
		}
		if res != c.Out {
			t.Errorf("res:%s != out:%s, code: %s", res, c.Out, c.In)
		}
	}

	invalidCodes := []struct {
		In  string
		Err string
	}{
		{
			"source%3Dgoogle.commmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"code longer than 200 characters",
		},
		{
			"medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",
			"code is missing keys",
		},
	}
	for _, c := range invalidCodes {
		_, err := validateAttributionCode(c.In)
		if err.Error() != c.Err {
			t.Errorf("err: %v != expected: %v", err, c.Err)
		}
	}

}
