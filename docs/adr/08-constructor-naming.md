# Decision

Constructors should be named "New" when the package or the type uniquely determine what is being created. For instance in package "foo" name the constructor "New" instead of "NewFoo" because the "Foo" is already clear from the context.

# Rationale

Avoiding the tautology makes the code more readable.