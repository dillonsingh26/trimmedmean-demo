# Writes the four test samples and R's trimmed means (data/r-results.csv)
# for the Go-vs-R comparison; run from the repository root with
# Rscript r/make-samples.R
# The 1001-value samples test R's floor() rule: 1001 * 0.05 = 50.05, so
# R and Go must both drop 50 values from each end.

set.seed(431)

# Shuffle so the outliers are not grouped at the end of the file.
shuffle <- function(x) sample(x, length(x))

samples <- list(
  "integers-100" = shuffle(c(
    round(rnorm(94, mean = 50, sd = 10)),
    sample(150:300, 4),   # high outliers
    sample(-60:-20, 2)    # low outliers
  )),
  "integers-1001" = shuffle(c(
    round(rnorm(951, mean = 50, sd = 10)),
    sample(150:300, 35),
    sample(-60:-20, 15)
  )),
  "floats-100" = shuffle(c(
    rnorm(95, mean = 100, sd = 10),
    rnorm(5, mean = 200, sd = 1)
  )),
  # same contaminated mixture as run-bootstrap-robust.R, with 5% outliers
  "floats-1001" = ifelse(runif(1001) < 0.05,
                         rnorm(1001, mean = 200, sd = 1),
                         rnorm(1001, mean = 100, sd = 10))
)

# One value per line under a "value" header. Floats are written with 17
# significant digits, enough to read back the exact same double in Go.
for (name in names(samples)) {
  x <- samples[[name]]
  text <- if (startsWith(name, "integers")) as.character(as.integer(x)) else sprintf("%.17g", x)
  writeLines(c("value", text), file.path("data", paste0(name, ".csv")))
}

# Compute R's results from the files just written, so R and Go read
# exactly the same numbers.
trims <- c(0, 0.05, 0.10, 0.20)
results <- do.call(rbind, lapply(names(samples), function(name) {
  x <- read.csv(file.path("data", paste0(name, ".csv")))$value
  data.frame(sample = name,
             trim = trims,
             r_value = sprintf("%.17g", sapply(trims, function(p) mean(x, trim = p))))
}))
write.csv(results, "data/r-results.csv", quote = FALSE, row.names = FALSE)

cat("R", R.version$major, ".", R.version$minor, " wrote ", nrow(results),
    " results to data/r-results.csv\n", sep = "")
print(results, row.names = FALSE)
