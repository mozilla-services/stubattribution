import unittest

from stub_attribution import webapp


class TestWebapp(unittest.TestCase):
    def test_validate_attribution_code_valid(self):
        cases = [
            ["source%3Dgoogle.com%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",  # NOQA
             "source=google.com&medium=organic&campaign=%28not+set%29&content=%28not+set%29"],  # NOQA
        ]

        for c in cases:
            code = webapp.validate_attribution_code(c[0])
            self.assertEqual(code, c[1])

    def test_validate_attribution_code_invalid(self):
        cases = [
            ["medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",  # NOQA
             "^code contains invalid or is missing keys$"],
            ["source%3Dgoogle.commmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm%26medium%3Dorganic%26campaign%3D(not%20set)%26content%3D(not%20set)",  # NOQA
             "^code longer than 200 characters$"],
        ]

        for c in cases:
            with self.assertRaisesRegexp(webapp.ValidationException, c[1]):
                webapp.validate_attribution_code(c[0])


if __name__ == '__main__':
    unittest.main()
