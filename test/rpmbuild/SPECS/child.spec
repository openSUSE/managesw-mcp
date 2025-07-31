Name:           child
Version:        1.0
Release:        1
Summary:        A child package
License:        MIT
Prefix:         /usr
Requires:       base
Recommends:     grandchild
AutoReqProv:    no

%description
This is a child package that depends on base.

%install
mkdir -p %{buildroot}/%{_prefix}/bin
cat << EOF > %{buildroot}/%{_prefix}/bin/child_command
#!/bin/sh
echo "Hello from child"
EOF
chmod +x %{buildroot}/%{_prefix}/bin/child_command

%files
%{_prefix}/bin/child_command