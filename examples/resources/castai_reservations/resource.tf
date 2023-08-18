resource "castai_reservations" "test" {
  reservations_csv = file("./reservations.csv")
}