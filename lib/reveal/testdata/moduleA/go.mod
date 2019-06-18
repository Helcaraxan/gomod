module moduleA

go 1.12

require (
    originalB v1.0.0
    originalC v1.0.0
    originalD v1.0.0
)

replace originalD => ./overrideD

replace (
    originalB => overrideB v1.0.0
    originalC => ./overrideC
)
