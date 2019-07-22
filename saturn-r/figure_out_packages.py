packages = []
with open("packages.txt") as f:
    for line in f:
        packages.append(line.strip())

apt_packages = {}

with open("rcran-apt.txt") as f:
    for line in f:
        if line.startswith('r-cran'):
            package = line.split('/')[0]
            rpackage = package.split('r-cran-')[-1]
            apt_packages[rpackage] = package

to_install_apt_package = []
to_install_r_pacakge = []

for p in packages:
    if p.lower() in apt_packages:
        to_install_apt_package.append(apt_packages[p.lower()])
    else:
        to_install_r_pacakge.append(p)

print ('APT')
for p in to_install_apt_package:
    print(p)

print ('R')
for p in to_install_r_pacakge:
    print(p)
