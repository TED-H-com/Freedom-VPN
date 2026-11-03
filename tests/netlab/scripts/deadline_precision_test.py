"""TEST ONLY offline D-104 regression for F16-01; no network or clock sleeps."""
import unittest

from deadline_precision import check_precision


class DeadlinePrecisionTests(unittest.TestCase):
    def test_D104_F16_01_before_fail_after_pass(self):
        values = [(1001, 699), (1002, 700), (1003, 701), (1004, 702),
                  (1005, 703), (1006, 704), (1007, 706), (1008, 707)]
        # Exact historical line 35: fails on the sixth adjacent step.
        with self.assertRaises(AssertionError):
            assert all(values[i+1][1]-values[i][1] == 1 for i in range(7)), values
        check_precision(values)

    def test_D104_coarser_quantum_fails(self):
        for quantum in (2, 4, 10, 1000):
            with self.subTest(quantum=quantum):
                values = [((1000+i)//quantum*quantum, (700+i)//quantum*quantum)
                          for i in range(1, 9)]
                with self.assertRaises(AssertionError):
                    check_precision(values)

    def test_D104_dump_clock_step_is_not_quantization(self):
        # One extra dump tick can shrink an adjacent step to zero; nft sorts
        # the hash traversal, so age need not be monotone in printed order.
        values = [(1000+i, 700+i-(3 if i == 4 else 2)) for i in range(1, 9)]
        check_precision(values)

    def test_D104_expiration_remains_live_and_bounded(self):
        for expires in (0, -1, 702, 1001):
            with self.subTest(expires=expires):
                values = [(1000+i, 700+i) for i in range(1, 9)]
                values[0] = (1001, expires)
                with self.assertRaises(AssertionError):
                    check_precision(values)

    def test_D104_all_exact_timeouts_required(self):
        values = [(1000+i, 698+i) for i in range(1, 9)]
        for bad in (values[:-1], values+[values[-1]], list(reversed(values)),
                    [(1002, 699)]+values[1:]):
            with self.subTest(values=bad), self.assertRaises(AssertionError):
                check_precision(bad)


if __name__ == '__main__':
    unittest.main()
